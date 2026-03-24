package modulegen

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclwrite"
	hclhelper "github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/zclconf/go-cty/cty"
)

//go:embed templates/import_guide.tmpl
var importGuideTmplContent string

func RenderImportBlocks(importBlocks []*ImportBlock) []byte {
	f := hclwrite.NewEmptyFile()
	body := f.Body()
	for i, importBlock := range importBlocks {
		if i > 0 {
			body.AppendNewline()
		}
		b := body.AppendNewBlock("import", nil)
		b.Body().SetAttributeValue("id", cty.StringVal(importBlock.ID))
		b.Body().SetAttributeRaw("to", hclhelper.TokensFromExpr(importBlock.To))
	}
	return f.Bytes()
}

func RenderModuleBlocks(moduleBlocks []*ModuleBlock) []byte {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	for i, moduleBlock := range moduleBlocks {
		if i > 0 {
			body.AppendNewline()
		}
		blockBody := body.AppendNewBlock("module", []string{moduleBlock.Name}).Body()
		blockBody.SetAttributeValue("source", cty.StringVal(moduleBlock.Source))
		blockBody.SetAttributeValue("version", cty.StringVal(moduleBlock.Version))
		for _, attr := range moduleBlock.Attributes {
			appendAttr(blockBody, attr)
		}
	}

	return f.Bytes()
}

type RenderedVariables struct {
	Blocks      []byte // For variables.tf
	Definitions []byte // For terraform.tfvars
}

func RenderVariables(variables []*Variable) RenderedVariables {
	blocksFile := hclwrite.NewEmptyFile()
	blocksBody := blocksFile.Body()
	defsFile := hclwrite.NewEmptyFile()
	defsBody := defsFile.Body()

	for i, v := range variables {
		if i > 0 {
			blocksBody.AppendNewline()
		}
		blockBody := blocksBody.AppendNewBlock("variable", []string{v.Name}).Body()
		blockBody.SetAttributeRaw("type", hclhelper.TokensFromExpr(typeexpr.TypeString(v.Type)))
		if v.Description != "" {
			blockBody.SetAttributeValue("description", cty.StringVal(v.Description))
		}

		defsBody.SetAttributeValue(v.Name, v.Value)
	}

	return RenderedVariables{Blocks: blocksFile.Bytes(), Definitions: defsFile.Bytes()}
}

type ImportGuideData struct {
	ModuleTypes []ModuleType
}

func RenderVersionsAndProviders(terraformVersion Version, providers []ProviderInfo) []byte {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	tfBlock := body.AppendNewBlock("terraform", nil)
	tfBody := tfBlock.Body()
	tfBody.SetAttributeValue("required_version", cty.StringVal(terraformVersion.String()))

	rpBody := tfBody.AppendNewBlock("required_providers", nil).Body()
	for _, p := range providers {
		nested := hclwrite.NewEmptyFile().Body()
		nested.SetAttributeValue("source", cty.StringVal(ProviderSource(p.ProviderType)))
		nested.SetAttributeValue("version", cty.StringVal(p.Version.String()))
		rpBody.SetAttributeRaw(string(p.ProviderType), hclhelper.TokensObject(nested))
	}

	for _, p := range providers {
		body.AppendNewline()
		// TODO support provider attributes, e.g. GCP project_id
		body.AppendNewBlock("provider", []string{string(p.ProviderType)})
	}

	return f.Bytes()
}

var importGuideTemplate = template.Must(
	template.New("import-guide").Parse(importGuideTmplContent),
)

func RenderImportGuide(data ImportGuideData) []byte {
	var buf bytes.Buffer
	if err := importGuideTemplate.Execute(&buf, data); err != nil {
		// Template execution only fails if the writer errors, not possible with a bytes.Buffer.
		panic(fmt.Sprintf("import guide template execution failed: %v", err))
	}
	return buf.Bytes()
}

// appendAttr writes a single Attribute into the body.
func appendAttr(body *hclwrite.Body, attr Attribute) {
	if attr.Comment != nil {
		body.AppendUnstructuredTokens(hclhelper.TokensComment(*attr.Comment))
	}
	switch {
	case attr.Literal != nil:
		body.SetAttributeValue(attr.Name, *attr.Literal)
	case attr.Variable != nil:
		body.SetAttributeTraversal(attr.Name, hcl.Traversal{
			hcl.TraverseRoot{Name: "var"},
			hcl.TraverseAttr{Name: attr.Variable.Name},
		})
	case len(attr.NestedInputs) > 0:
		nested := hclwrite.NewEmptyFile().Body()
		for _, child := range attr.NestedInputs {
			appendAttr(nested, child)
		}
		body.SetAttributeRaw(attr.Name, hclhelper.TokensObject(nested))
		return
	}
}
