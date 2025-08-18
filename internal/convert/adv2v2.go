package convert

import (
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/zclconf/go-cty/cty"
)

// AdvancedClusterToV2 transforms all mongodbatlas_advanced_cluster resource definitions in a
// Terraform configuration file from SDKv2 schema to TPF (Terraform Plugin Framework) schema.
// All other resources and data sources are left untouched.
// TODO: Not implemented yet.
func AdvancedClusterToV2(config []byte) ([]byte, error) {
	parser, err := hcl.GetParser(config)
	if err != nil {
		return nil, err
	}
	parserb := parser.Body()
	for _, block := range parserb.Blocks() {
		updated, err := updateResource(block)
		if err != nil {
			return nil,
				err
		}
		if updated {
			blockb := block.Body()
			blockb.AppendNewline()
			hcl.AppendComment(blockb, commentUpdatedBy)
		}
	}
	return parser.Bytes(), nil
}

func updateResource(resource *hclwrite.Block) (bool, error) {
	if resource.Type() != resourceType || getResourceName(resource) != advCluster {
		return false, nil
	}
	resourceb := resource.Body()
	// TODO: Remove: convertAttrs("replication_specs", resourceb, true, getReplicationSpecs)
	if err := convertRepSpecs(resourceb); err != nil {
		return false, err
	}
	if err := fillTagsLabelsOpt(resourceb, nTags); err != nil {
		return false, err
	}
	if err := fillTagsLabelsOpt(resourceb, nLabels); err != nil {
		return false, err
	}
	fillBlockOpt(resourceb, nAdvConf)
	fillBlockOpt(resourceb, nBiConnector)
	fillBlockOpt(resourceb, nPinnedFCV)
	fillBlockOpt(resourceb, nTimeouts)
	return true, nil
}

func convertRepSpecs(resourceb *hclwrite.Body) error {
	block := resourceb.FirstMatchingBlock(nRepSpecs, nil)
	if block == nil {
		return nil
	}
	resourceb.RemoveBlock(block)
	if err := convertConfig(block.Body()); err != nil {
		return err
	}
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArraySingle(block.Body()))
	return nil
}

func convertConfig(repSpecs *hclwrite.Body) error {
	block := repSpecs.FirstMatchingBlock(nConfig, nil)
	if block == nil {
		return nil
	}
	repSpecs.RemoveBlock(block)
	fillBlockOpt(block.Body(), nElectableSpecs)
	repSpecs.SetAttributeRaw(nConfig, hcl.TokensArraySingle(block.Body()))
	return nil
}

func TodoConvertAttrs(name string, writeBody *hclwrite.Body, isList bool, getOneAttr func(*hclsyntax.Body) cty.Value) {
	var vals []cty.Value
	for {
		match := writeBody.FirstMatchingBlock(name, nil)
		if match == nil {
			break
		}
		vals = append(vals, getOneAttr(GetBlockBody(match)))
		writeBody.RemoveBlock(match) // RemoveBlock doesn't remove newline just after the block so an extra line is added
	}
	if len(vals) == 0 {
		return
	}
	if isList {
		writeBody.SetAttributeValue(name, cty.TupleVal(vals))
	} else {
		// TODO assert.Len(t, vals, 1, "can be only one of %s", name)
		writeBody.SetAttributeValue(name, vals[0])
	}
}

func TodoKeyValueAttrs(name string, writeBody *hclwrite.Body) {
	vals := make(map[string]cty.Value)
	for {
		match := writeBody.FirstMatchingBlock(name, nil)
		if match == nil {
			break
		}
		attrs := HclGetAttrVal(GetBlockBody(match))
		key := attrs.GetAttr("key")
		value := attrs.GetAttr("value")
		vals[key.AsString()] = value
		writeBody.RemoveBlock(match) // RemoveBlock doesn't remove newline just after the block so an extra line is added
	}
	if len(vals) > 0 {
		writeBody.SetAttributeValue(name, cty.ObjectVal(vals))
	}
}

func GetReplicationSpecs(body *hclsyntax.Body) cty.Value {
	const name = "region_configs"
	var vals []cty.Value
	for _, block := range body.Blocks {
		// TODO assert.Equal(t, name, block.Type, "unexpected block type: %s", block.Type)
		vals = append(vals, HclGetAttrVal(block.Body))
	}
	attributeValues := map[string]cty.Value{
		name: cty.TupleVal(vals),
	}
	HclAddAttributes(body, attributeValues)
	return cty.ObjectVal(attributeValues)
}

func HclGetAttrVal(body *hclsyntax.Body) cty.Value {
	ret := make(map[string]cty.Value)
	HclAddAttributes(body, ret)
	for _, block := range body.Blocks {
		ret[block.Type] = HclGetAttrVal(block.Body)
	}
	return cty.ObjectVal(ret)
}

func HclAddAttributes(body *hclsyntax.Body, ret map[string]cty.Value) {
	for name, attr := range body.Attributes {
		val, diags := attr.Expr.Value(nil)
		// TODO require.False(t, diags.HasErrors(), "failed to parse attribute %s: %s", name, diags.Error())
		_ = diags // TODO remove
		ret[name] = val
	}
}

func GetBlockBody(block *hclwrite.Block) *hclsyntax.Body {
	parser, diags := hclparse.NewParser().ParseHCL(block.Body().BuildTokens(nil).Bytes(), "")
	// TODO require.False(t, diags.HasErrors(), "failed to parse block: %s", diags.Error())

	body, ok := parser.Body.(*hclsyntax.Body)
	// TODO require.True(t, ok, "unexpected *hclsyntax.Body type: %T", parser.Body)

	_, _ = ok, diags // TODO remove
	return body
}
