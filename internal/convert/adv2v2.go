package convert

import (
	"fmt"
	"slices"

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
			return nil, err
		}
		if updated { // If the resource was converted, add a comment at the end so user knows the resource was updated
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
	if hasExpectedBlocksAsAttributes(resourceb) {
		return false, nil
	}
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
	var configs []*hclwrite.Body
	for {
		block := repSpecs.FirstMatchingBlock(nConfig, nil)
		if block == nil {
			break
		}
		repSpecs.RemoveBlock(block)
		blockb := block.Body()
		fillBlockOpt(blockb, nElectableSpecs)
		fillBlockOpt(blockb, nReadOnlySpecs)
		fillBlockOpt(blockb, nAnalyticsSpecs)
		fillBlockOpt(blockb, nAutoScaling)
		fillBlockOpt(blockb, nAnalyticsAutoScaling)
		configs = append(configs, blockb)
	}
	if len(configs) == 0 {
		return fmt.Errorf("replication_specs must have at least one region_configs")
	}
	repSpecs.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
	return nil
}

// hasExpectedBlocksAsAttributes checks if any of the expected block names
// exist as attributes in the resource body. In that case conversion is not done
// as advanced cluster is not in a valid SDKv2 configuration.
func hasExpectedBlocksAsAttributes(resourceb *hclwrite.Body) bool {
	expectedBlocks := []string{
		nRepSpecs,
		nTags,
		nLabels,
		nAdvConf,
		nBiConnector,
		nPinnedFCV,
		nTimeouts,
	}
	for name := range resourceb.Attributes() {
		if slices.Contains(expectedBlocks, name) {
			return true
		}
	}
	return false
}
