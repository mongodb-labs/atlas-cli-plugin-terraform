package modulegen

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
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
		if len(importBlock.ForEach) > 0 {
			b.Body().SetAttributeRaw("for_each", hclhelper.TokensFromExpr(
				fmt.Sprintf("toset([%s])", `"`+strings.Join(importBlock.ForEach, `", "`)+`"`),
			))
			b.Body().SetAttributeRaw("id", hclhelper.TokensFromExpr(fmt.Sprintf("%q", importBlock.ID)))
		} else {
			b.Body().SetAttributeValue("id", cty.StringVal(importBlock.ID))
		}
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
		blockBody.SetAttributeValue("version", versionValue(moduleBlock.Version))
		body.AppendNewline()
		for _, attr := range moduleBlock.Attributes {
			appendAttr(blockBody, &attr)
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
		if v.DefaultValue != nil {
			blockBody.SetAttributeValue("default", *v.DefaultValue)
		}

		defsBody.SetAttributeValue(v.Name, v.Value)
	}

	return RenderedVariables{Blocks: blocksFile.Bytes(), Definitions: defsFile.Bytes()}
}

func RenderVersionsAndProviders(tfVersion Version, providers []*ProviderInfo) []byte {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	tfBlock := body.AppendNewBlock("terraform", nil)
	tfBody := tfBlock.Body()
	tfBody.SetAttributeValue("required_version", versionValue(tfVersion))

	rpBody := tfBody.AppendNewBlock("required_providers", nil).Body()
	for _, provider := range providers {
		nested := hclwrite.NewEmptyFile().Body()
		nested.SetAttributeValue("source", cty.StringVal(provider.Source))
		nested.SetAttributeValue("version", versionValue(provider.Version))
		rpBody.SetAttributeRaw(provider.Name, hclhelper.TokensObject(nested))
	}

	for _, provider := range providers {
		body.AppendNewline()
		pBody := body.AppendNewBlock("provider", []string{provider.Name}).Body()
		for _, attr := range provider.Attributes {
			appendAttr(pBody, &attr)
		}
	}

	return f.Bytes()
}

type ImportGuideData struct {
	ModuleTypes []ModuleType
}

func RenderImportGuide(data ImportGuideData) []byte {
	var buf bytes.Buffer
	tmpl := template.Must(template.New("import-guide").Parse(importGuideTmplContent))
	if err := tmpl.Execute(&buf, data); err != nil {
		// Template execution only fails if the writer errors, not possible with a bytes.Buffer.
		panic(fmt.Sprintf("import guide template execution failed: %v", err))
	}
	return buf.Bytes()
}

// Intentionally not using v.String() here to avoid unexpected regressions
func versionValue(v Version) cty.Value {
	if v.Operator == "" {
		return cty.StringVal(fmt.Sprintf("%d.%d", v.Major, v.Minor))
	}
	return cty.StringVal(fmt.Sprintf("%s %d.%d", v.Operator, v.Major, v.Minor))
}

// appendAttr writes a single Attribute into the body.
func appendAttr(body *hclwrite.Body, attr *Attribute) {
	// Skipping attributes matching the module default Value. This can be made user-configurable for an explicit mode.
	if attr.IsDefaultValue {
		return
	}
	if attr.Comment != nil {
		body.AppendUnstructuredTokens(hclhelper.TokensComment(*attr.Comment))
	}
	switch {
	case attr.Value.Literal != nil:
		body.SetAttributeValue(attr.Name, *attr.Value.Literal)
	case attr.Value.Variable != nil:
		body.SetAttributeTraversal(attr.Name, hcl.Traversal{
			hcl.TraverseRoot{Name: "var"},
			hcl.TraverseAttr{Name: attr.Value.Variable.Name},
		})
	case len(attr.Value.Object) > 0:
		body.SetAttributeRaw(attr.Name, hclhelper.TokensObject(renderObjectBody(attr.Value.Object)))
	case len(attr.Value.ObjectList) > 0:
		bodies := make([]*hclwrite.Body, len(attr.Value.ObjectList))
		for i, item := range attr.Value.ObjectList {
			bodies[i] = renderObjectBody(item)
		}
		body.SetAttributeRaw(attr.Name, hclhelper.TokensArray(bodies))
	}
}

func renderObjectBody(object []Attribute) *hclwrite.Body {
	body := hclwrite.NewEmptyFile().Body()
	for _, child := range object {
		appendAttr(body, &child)
	}
	return body
}
