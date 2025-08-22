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
	fillAdvConfigOpt(resourceb)
	fillBlockOpt(resourceb, nBiConnector)
	fillBlockOpt(resourceb, nPinnedFCV)
	fillBlockOpt(resourceb, nTimeouts)
	return true, nil
}

func convertRepSpecs(resourceb *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
	// Handle dynamic blocks for replication_specs
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs)
	if err != nil {
		return err
	}
	if dSpec.IsPresent() {
		return convertDynamicRepSpecs(resourceb, dSpec, diskSizeGB)
	}

	// Collect all replication_specs blocks first
	var repSpecBlocks []*hclwrite.Block
	for {
		block := resourceb.FirstMatchingBlock(nRepSpecs, nil)
		if block == nil {
			break
		}
		resourceb.RemoveBlock(block)
		repSpecBlocks = append(repSpecBlocks, block)
	}

	if len(repSpecBlocks) == 0 {
		return fmt.Errorf("must have at least one replication_specs")
	}

	// Check if any replication_specs has a variable num_shards
	hasVariableNumShards := HasVariableNumShards(repSpecBlocks)

	if hasVariableNumShards {
		var concatParts []hclwrite.Tokens

		for _, block := range repSpecBlocks {
			blockb := block.Body()
			numShardsAttr := blockb.GetAttribute(nNumShards)
			blockb.RemoveAttribute(nNumShards)

			if err := convertConfig(blockb, diskSizeGB); err != nil {
				return err
			}

			tokens, err := ProcessNumShards(numShardsAttr, blockb)
			if err != nil {
				return err
			}
			concatParts = append(concatParts, tokens)
		}

		// Use concat to combine all parts
		if len(concatParts) > 1 {
			resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensFuncConcat(concatParts...))
		} else {
			resourceb.SetAttributeRaw(nRepSpecs, concatParts[0])
		}
	} else {
		// All num_shards are numeric or missing, use simple array
		var repSpecs []*hclwrite.Body
		for _, block := range repSpecBlocks {
			blockb := block.Body()
			numShardsAttr := blockb.GetAttribute(nNumShards)
			blockb.RemoveAttribute(nNumShards)

			if err := convertConfig(blockb, diskSizeGB); err != nil {
				return err
			}

			if numShardsAttr != nil {
				numShardsVal, _ := hcl.GetAttrInt(numShardsAttr, errNumShards)
				for range numShardsVal {
					repSpecs = append(repSpecs, blockb)
				}
			} else {
				// No num_shards, default to 1
				repSpecs = append(repSpecs, blockb)
			}
		}
		resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArray(repSpecs))
	}

	return nil
}

func convertDynamicRepSpecs(resourceb *hclwrite.Body, dSpec dynamicBlock, diskSizeGB hclwrite.Tokens) error {
	// Transform references from replication_specs.value.* to spec.*
	transformDynamicBlockReferences(dSpec.content.Body(), nRepSpecs, nSpec)

	// Check for dynamic region_configs within this dynamic replication_specs
	dConfig, err := getDynamicBlock(dSpec.content.Body(), nConfig)
	if err != nil {
		return err
	}
	if !dConfig.IsPresent() {
		dConfig, err = getDynamicBlock(dSpec.content.Body(), nConfigSrc)
		if err != nil {
			return err
		}
	}

	if dConfig.IsPresent() {
		// Handle nested dynamic block for region_configs
		return convertDynamicRepSpecsWithDynamicConfig(resourceb, dSpec, dConfig, diskSizeGB)
	}

	// Get num_shards from the dynamic block content
	numShardsAttr := dSpec.content.Body().GetAttribute(nNumShards)
	if numShardsAttr != nil {
		numShardsExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
		dSpec.content.Body().RemoveAttribute(nNumShards)

		// Convert region_configs inside the dynamic block
		if err := convertConfig(dSpec.content.Body(), diskSizeGB); err != nil {
			return err
		}

		// Create the for expression for the flattened replication_specs
		forExpr := fmt.Sprintf("for %s in %s : [\n    for i in range(%s) : ",
			nSpec, hcl.GetAttrExpr(dSpec.forEach), numShardsExpr)
		tokens := hcl.TokensFromExpr(forExpr)
		tokens = append(tokens, hcl.TokensObject(dSpec.content.Body())...)
		tokens = append(tokens, hcl.TokensFromExpr("\n  ]\n]")...)

		resourceb.RemoveBlock(dSpec.block)
		resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensFuncFlatten(tokens))
		return nil
	}
	// No num_shards, default to 1
	dSpec.content.Body().RemoveAttribute(nNumShards)

	// Convert region_configs inside the dynamic block
	if err := convertConfig(dSpec.content.Body(), diskSizeGB); err != nil {
		return err
	}

	// Create the for expression without num_shards
	forExpr := fmt.Sprintf("for %s in %s :", nSpec, hcl.GetAttrExpr(dSpec.forEach))
	tokens := hcl.TokensFromExpr(forExpr)
	tokens = append(tokens, hcl.TokensObject(dSpec.content.Body())...)
	tokens = hcl.EncloseBracketsNewLines(tokens)

	resourceb.RemoveBlock(dSpec.block)
	resourceb.SetAttributeRaw(nRepSpecs, tokens)
	return nil
}

// Helper function to process blocks for region configs
func processRegionConfigBlocks(targetBody *hclwrite.Body, blocks []*hclwrite.Block) {
	for _, block := range blocks {
		blockType := block.Type()
		blockFile := hclwrite.NewEmptyFile()
		blockBody := blockFile.Body()

		// Copy all attributes in deterministic order
		attrs := block.Body().Attributes()
		var names []string
		for name := range attrs {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			attr := attrs[name]
			blockBody.SetAttributeRaw(name, hcl.TokensFromExpr(hcl.GetAttrExpr(attr)))
		}

		targetBody.SetAttributeRaw(blockType, hcl.TokensObject(blockBody))
	}
}

func convertDynamicRepSpecsWithDynamicConfig(resourceb *hclwrite.Body, dSpec, dConfig dynamicBlock,
	diskSizeGB hclwrite.Tokens) error {
	// Get the block name from the dynamic block
	configBlockName := getResourceName(dConfig.block)

	// Get num_shards expression
	numShardsAttr := dSpec.content.Body().GetAttribute(nNumShards)
	if numShardsAttr != nil {
		numShardsExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)

		// Transform references in place for the dynamic config content
		transformDynamicBlockReferencesRecursive(dConfig.content.Body(), configBlockName, nRegion)
		// Also transform outer references (with deterministic ordering)
		attrs := dConfig.content.Body().Attributes()
		var attrNames []string
		for name := range attrs {
			attrNames = append(attrNames, name)
		}
		slices.Sort(attrNames)
		for _, name := range attrNames {
			attr := attrs[name]
			expr := hcl.GetAttrExpr(attr)
			expr = replaceDynamicBlockReferences(expr, nRepSpecs, nSpec)
			dConfig.content.Body().SetAttributeRaw(name, hcl.TokensFromExpr(expr))
		}
		for _, block := range dConfig.content.Body().Blocks() {
			blockAttrs := block.Body().Attributes()
			var blockAttrNames []string
			for name := range blockAttrs {
				blockAttrNames = append(blockAttrNames, name)
			}
			slices.Sort(blockAttrNames)
			for _, name := range blockAttrNames {
				attr := blockAttrs[name]
				expr := hcl.GetAttrExpr(attr)
				expr = replaceDynamicBlockReferences(expr, nRepSpecs, nSpec)
				block.Body().SetAttributeRaw(name, hcl.TokensFromExpr(expr))
			}
		}

		// Build the expression using HCL functions
		// Use standardized property name (region_configs) instead of the actual for_each collection
		configForEach := fmt.Sprintf("%s.%s", nSpec, nConfig)

		// Create the inner region_configs body
		regionConfigFile := hclwrite.NewEmptyFile()
		regionConfigBody := regionConfigFile.Body()

		// Copy all attributes in deterministic order
		configAttrs := dConfig.content.Body().Attributes()
		var configAttrNames []string
		for name := range configAttrs {
			configAttrNames = append(configAttrNames, name)
		}
		slices.Sort(configAttrNames)
		for _, name := range configAttrNames {
			attr := configAttrs[name]
			regionConfigBody.SetAttributeRaw(name, hcl.TokensFromExpr(hcl.GetAttrExpr(attr)))
		}

		// Add all blocks generically as objects
		processRegionConfigBlocks(regionConfigBody, dConfig.content.Body().Blocks())

		// Build the region_configs for expression
		regionForExpr := fmt.Sprintf("for %s in %s :", nRegion, configForEach)
		regionTokens := hcl.TokensFromExpr(regionForExpr)
		regionTokens = append(regionTokens, hcl.TokensObject(regionConfigBody)...)

		// Create the replication spec body
		repSpecFile := hclwrite.NewEmptyFile()
		repSpecBody := repSpecFile.Body()

		if zoneNameAttr := dSpec.content.Body().GetAttribute(nZoneName); zoneNameAttr != nil {
			zoneNameExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
			repSpecBody.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
		}

		repSpecBody.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))

		// Build the inner for expression with range
		innerForExpr := fmt.Sprintf("for i in range(%s) :", numShardsExpr)
		innerTokens := hcl.TokensFromExpr(innerForExpr)
		innerTokens = append(innerTokens, hcl.TokensObject(repSpecBody)...)

		// Build the outer for expression
		outerForExpr := fmt.Sprintf("for %s in %s : ", nSpec, hcl.GetAttrExpr(dSpec.forEach))
		outerTokens := hcl.TokensFromExpr(outerForExpr)
		outerTokens = append(outerTokens, hcl.EncloseBracketsNewLines(innerTokens)...)

		// Apply flatten to the entire expression
		tokens := hcl.TokensFuncFlatten(outerTokens)

		resourceb.RemoveBlock(dSpec.block)
		resourceb.SetAttributeRaw(nRepSpecs, tokens)
		return nil
	}
	// No num_shards, handle like without nested shards
	return convertDynamicRepSpecsWithoutNumShards(resourceb, dSpec, dConfig, diskSizeGB, configBlockName)
}

// Helper function to add attributes with transformation
func addAttributesWithTransform(targetBody *hclwrite.Body, sourceAttrs map[string]*hclwrite.Attribute,
	configBlockName string) {
	// Sort attribute names for deterministic output
	var names []string
	for name := range sourceAttrs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		attr := sourceAttrs[name]
		expr := hcl.GetAttrExpr(attr)
		// First replace nested dynamic block references
		expr = replaceDynamicBlockReferences(expr, configBlockName, nRegion)
		// Then replace outer dynamic block references
		expr = replaceDynamicBlockReferences(expr, nRepSpecs, nSpec)
		targetBody.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
}

func convertDynamicRepSpecsWithoutNumShards(resourceb *hclwrite.Body, dSpec, dConfig dynamicBlock,
	diskSizeGB hclwrite.Tokens, configBlockName string) error {
	// Get zone_name if present
	repSpecFile := hclwrite.NewEmptyFile()
	repSpecb := repSpecFile.Body()

	if zoneNameAttr := dSpec.content.Body().GetAttribute(nZoneName); zoneNameAttr != nil {
		zoneNameExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
		repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
	}

	// Create config content with transformed references
	configFile := hclwrite.NewEmptyFile()
	configb := configFile.Body()

	// Copy and transform attributes
	addAttributesWithTransform(configb, dConfig.content.Body().Attributes(), configBlockName)

	// Process blocks and transform their references
	for _, block := range dConfig.content.Body().Blocks() {
		newBlock := configb.AppendNewBlock(block.Type(), block.Labels())
		newBlockb := newBlock.Body()
		addAttributesWithTransform(newBlockb, block.Body().Attributes(), configBlockName)
	}

	// Process specs
	fillSpecOpt(configb, nElectableSpecs, diskSizeGB)
	fillSpecOpt(configb, nReadOnlySpecs, diskSizeGB)
	fillSpecOpt(configb, nAnalyticsSpecs, diskSizeGB)
	fillSpecOpt(configb, nAutoScaling, nil)
	fillSpecOpt(configb, nAnalyticsAutoScaling, nil)

	// Build the nested for expression for region_configs
	// Use standardized property name (region_configs) instead of the actual for_each collection
	configForEach := fmt.Sprintf("%s.%s", nSpec, nConfig)

	// Build the region_configs for expression
	regionForExpr := fmt.Sprintf("for %s in %s :", nRegion, configForEach)
	regionTokens := hcl.TokensFromExpr(regionForExpr)
	regionTokens = append(regionTokens, hcl.TokensObject(configb)...)

	repSpecb.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))

	// Build the for expression as an array wrapped in flatten
	// Format: flatten([for spec in ... : [ { ... } ] ])
	forExpr := fmt.Sprintf("for %s in %s : [\n      ", nSpec, hcl.GetAttrExpr(dSpec.forEach))
	innerTokens := hcl.TokensFromExpr(forExpr)
	innerTokens = append(innerTokens, hcl.TokensObject(repSpecb)...)
	innerTokens = append(innerTokens, hcl.TokensFromExpr("\n    ]")...)

	// Apply flatten to the entire expression
	tokens := hcl.TokensFuncFlatten(innerTokens)

	resourceb.RemoveBlock(dSpec.block)
	resourceb.SetAttributeRaw(nRepSpecs, tokens)
	return nil
}

func convertConfig(repSpecs *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
	// Check for dynamic region_configs block (can be either "region_configs" or "regions_config")
	dConfig, err := getDynamicBlock(repSpecs, nConfig)
	if err != nil {
		return err
	}
	if !dConfig.IsPresent() {
		dConfig, err = getDynamicBlock(repSpecs, nConfigSrc)
		if err != nil {
			return err
		}
	}

	if dConfig.IsPresent() {
		// Handle dynamic region_configs
		return convertDynamicConfig(repSpecs, dConfig, diskSizeGB)
	}

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

func convertDynamicConfig(repSpecs *hclwrite.Body, dConfig dynamicBlock, diskSizeGB hclwrite.Tokens) error {
	// Get the block name from the dynamic block itself
	blockName := getResourceName(dConfig.block)

	// Transform the references in attributes and blocks
	transformDynamicBlockReferencesRecursive(dConfig.content.Body(), blockName, nRegion)

	// Process specs
	fillSpecOpt(dConfig.content.Body(), nElectableSpecs, diskSizeGB)
	fillSpecOpt(dConfig.content.Body(), nReadOnlySpecs, diskSizeGB)
	fillSpecOpt(dConfig.content.Body(), nAnalyticsSpecs, diskSizeGB)
	fillSpecOpt(dConfig.content.Body(), nAutoScaling, nil)
	fillSpecOpt(dConfig.content.Body(), nAnalyticsAutoScaling, nil)

	// Build the for expression
	forExpr := fmt.Sprintf("for %s in %s :", nRegion, hcl.GetAttrExpr(dConfig.forEach))
	tokens := hcl.TokensFromExpr(forExpr)
	tokens = append(tokens, hcl.TokensObject(dConfig.content.Body())...)
	tokens = hcl.EncloseBracketsNewLines(tokens)

	repSpecs.RemoveBlock(dConfig.block)
	repSpecs.SetAttributeRaw(nConfig, tokens)
	return nil
}

func transformDynamicBlockReferencesRecursive(body *hclwrite.Body, blockName, varName string) {
	// Transform attributes in deterministic order
	attrs := body.Attributes()
	var names []string
	for name := range attrs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		attr := attrs[name]
		expr := replaceDynamicBlockReferences(hcl.GetAttrExpr(attr), blockName, varName)
		body.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
	// Transform nested blocks
	for _, block := range body.Blocks() {
		transformDynamicBlockReferencesRecursive(block.Body(), blockName, varName)
	}
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
		nAdvConfig,
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
