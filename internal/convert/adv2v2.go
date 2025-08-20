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
	diskSizeGB, _ := hcl.PopAttr(resourceb, nDiskSizeGB, errRoot) // ok to fail as it's optional
	if err := convertRepSpecs(resourceb, diskSizeGB); err != nil {
		return false, err
	}
	if err := fillTagsLabelsOpt(resourceb, nTags); err != nil {
		return false, err
	}
	if err := fillTagsLabelsOpt(resourceb, nLabels); err != nil {
		return false, err
	}
	fillAdvConfigOpt(resourceb, nAdvConf)
	fillBlockOpt(resourceb, nBiConnector)
	fillBlockOpt(resourceb, nPinnedFCV)
	fillBlockOpt(resourceb, nTimeouts)
	return true, nil
}

func convertRepSpecs(resourceb *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
	var repSpecs []*hclwrite.Body
	for {
		block := resourceb.FirstMatchingBlock(nRepSpecs, nil)
		if block == nil {
			break
		}
		resourceb.RemoveBlock(block)
		blockb := block.Body()
		numShardsVal := 1 // default to 1 if num_shards not present
		if numShardsAttr := blockb.GetAttribute(nNumShards); numShardsAttr != nil {
			var err error
			if numShardsVal, err = hcl.GetAttrInt(numShardsAttr, errNumShards); err != nil {
				return err
			}
			blockb.RemoveAttribute(nNumShards)
		}
		if err := convertConfig(blockb, diskSizeGB); err != nil {
			return err
		}
		for range numShardsVal {
			repSpecs = append(repSpecs, blockb)
		}
	}
	if len(repSpecs) == 0 {
		return fmt.Errorf("must have at least one replication_specs")
	}
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArray(repSpecs))
	return nil
}

func convertConfig(repSpecs *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
	var configs []*hclwrite.Body
	for {
		block := repSpecs.FirstMatchingBlock(nConfig, nil)
		if block == nil {
			break
		}
		repSpecs.RemoveBlock(block)
		blockb := block.Body()
		fillSpecOpt(blockb, nElectableSpecs, diskSizeGB)
		fillSpecOpt(blockb, nReadOnlySpecs, diskSizeGB)
		fillSpecOpt(blockb, nAnalyticsSpecs, diskSizeGB)
		fillSpecOpt(blockb, nAutoScaling, nil)          // auto_scaling doesn't need disk_size_gb
		fillSpecOpt(blockb, nAnalyticsAutoScaling, nil) // analytics_auto_scaling doesn't need disk_size_gb
		configs = append(configs, blockb)
	}
	if len(configs) == 0 {
		return fmt.Errorf("replication_specs must have at least one region_configs")
	}
	repSpecs.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
	return nil
}

func fillSpecOpt(resourceb *hclwrite.Body, name string, diskSizeGBTokens hclwrite.Tokens) {
	block := resourceb.FirstMatchingBlock(name, nil)
	if block == nil {
		return
	}
	if diskSizeGBTokens != nil {
		blockb := block.Body()
		blockb.RemoveAttribute(nDiskSizeGB)
		blockb.SetAttributeRaw(nDiskSizeGB, diskSizeGBTokens)
	}
	fillBlockOpt(resourceb, name)
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
