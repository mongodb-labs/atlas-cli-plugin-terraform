package convert

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
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
