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
	if errDyn := checkDynamicBlock(resourceb); errDyn != nil {
		return false, errDyn
	}
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
	d, err := convertRepSpecsWithDynamicBlock(resourceb, diskSizeGB)
	if err != nil {
		return err
	}
	if d.IsPresent() {
		return nil
	}
	repSpecBlocks := collectBlocks(resourceb, nRepSpecs)
	if len(repSpecBlocks) == 0 {
		return fmt.Errorf("must have at least one replication_specs")
	}
	hasVariableShards := hasVariableNumShards(repSpecBlocks)
	var resultTokens []hclwrite.Tokens
	var resultBodies []*hclwrite.Body
	for _, block := range repSpecBlocks {
		blockb := block.Body()
		shardsAttr := blockb.GetAttribute(nNumShards)
		blockb.RemoveAttribute(nNumShards)
		if err := convertConfig(blockb, diskSizeGB); err != nil {
			return err
		}
		if hasVariableShards {
			resultTokens = append(resultTokens, processNumShards(shardsAttr, blockb))
			continue
		}
		numShardsVal := 1 // Default to 1 if num_shards is not set
		if shardsAttr != nil {
			numShardsVal, _ = hcl.GetAttrInt(shardsAttr, errNumShards)
		}
		for range numShardsVal {
			resultBodies = append(resultBodies, blockb)
		}
	}
	if hasVariableShards {
		resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensFuncConcat(resultTokens...))
	} else {
		resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArray(resultBodies))
	}
	return nil
}

func convertRepSpecsWithDynamicBlock(resourceb *hclwrite.Body, diskSizeGB hclwrite.Tokens) (dynamicBlock, error) {
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs)
	if err != nil {
		return dynamicBlock{}, err
	}
	if dSpec.IsPresent() {
		transformDynamicBlockReferences(dSpec.content.Body(), nRepSpecs, nSpec)
		dConfig, err := getDynamicBlock(dSpec.content.Body(), nConfig)
		if err != nil {
			return dynamicBlock{}, err
		}
		if !dConfig.IsPresent() {
			dConfig, err = getDynamicBlock(dSpec.content.Body(), nConfigSrc)
			if err != nil {
				return dynamicBlock{}, err
			}
		}
		if dConfig.IsPresent() {
			// Will inline convertDynamicRepSpecsWithDynamicConfig
			configBlockName := getResourceName(dConfig.block)
			numShardsAttr := dSpec.content.Body().GetAttribute(nNumShards)
			if numShardsAttr != nil {
				numShardsExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
				transform1 := func(expr string) string {
					return replaceDynamicBlockReferences(expr, configBlockName, nRegion)
				}
				transformAttributesSorted(dConfig.content.Body(), dConfig.content.Body().Attributes(), transform1)
				for _, block := range dConfig.content.Body().Blocks() {
					// Recursive call inlined
					transformAttributesSorted(block.Body(), block.Body().Attributes(), transform1)
				}
				transform2 := func(expr string) string {
					return replaceDynamicBlockReferences(expr, nRepSpecs, nSpec)
				}
				transformAttributesSorted(dConfig.content.Body(), dConfig.content.Body().Attributes(), transform2)
				for _, block := range dConfig.content.Body().Blocks() {
					transformAttributesSorted(block.Body(), block.Body().Attributes(), transform2)
				}
				regionConfigFile := hclwrite.NewEmptyFile()
				regionConfigBody := regionConfigFile.Body()
				copyAttributesSorted(regionConfigBody, dConfig.content.Body().Attributes())
				for _, block := range dConfig.content.Body().Blocks() {
					blockType := block.Type()
					blockFile := hclwrite.NewEmptyFile()
					blockBody := blockFile.Body()
					copyAttributesSorted(blockBody, block.Body().Attributes())
					if diskSizeGB != nil && (blockType == nElectableSpecs ||
						blockType == nReadOnlySpecs || blockType == nAnalyticsSpecs) {
						blockBody.SetAttributeRaw(nDiskSizeGB, diskSizeGB)
					}
					regionConfigBody.SetAttributeRaw(blockType, hcl.TokensObject(blockBody))
				}

				configForEach := fmt.Sprintf("%s.%s", nSpec, nConfig)
				regionForExpr := buildForExpr(nRegion, configForEach, false)
				regionTokens := hcl.TokensFromExpr(regionForExpr)
				regionTokens = append(regionTokens, hcl.TokensObject(regionConfigBody)...)

				repSpecFile := hclwrite.NewEmptyFile()
				repSpecBody := repSpecFile.Body()
				if zoneNameAttr := dSpec.content.Body().GetAttribute(nZoneName); zoneNameAttr != nil {
					zoneNameExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
					repSpecBody.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
				}
				repSpecBody.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))

				innerForExpr := buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false)
				innerTokens := hcl.TokensFromExpr(innerForExpr)
				innerTokens = append(innerTokens, hcl.TokensObject(repSpecBody)...)

				outerForExpr := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true)
				outerTokens := hcl.TokensFromExpr(outerForExpr)
				outerTokens = append(outerTokens, hcl.EncloseBracketsNewLines(innerTokens)...)

				tokens := hcl.TokensFuncFlatten(outerTokens)
				resourceb.RemoveBlock(dSpec.block)
				resourceb.SetAttributeRaw(nRepSpecs, tokens)
				return dSpec, nil
			}
			repSpecFile := hclwrite.NewEmptyFile()
			repSpecb := repSpecFile.Body()
			if zoneNameAttr := dSpec.content.Body().GetAttribute(nZoneName); zoneNameAttr != nil {
				zoneNameExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
				repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
			}
			configFile := hclwrite.NewEmptyFile()
			configb := configFile.Body()
			addAttributesWithTransform(configb, dConfig.content.Body().Attributes(), configBlockName)
			for _, block := range dConfig.content.Body().Blocks() {
				newBlock := configb.AppendNewBlock(block.Type(), block.Labels())
				newBlockb := newBlock.Body()
				addAttributesWithTransform(newBlockb, block.Body().Attributes(), configBlockName)
			}
			processAllSpecs(configb, diskSizeGB)
			configForEach := fmt.Sprintf("%s.%s", nSpec, nConfig)
			regionForExpr := buildForExpr(nRegion, configForEach, false)
			regionTokens := hcl.TokensFromExpr(regionForExpr)
			regionTokens = append(regionTokens, hcl.TokensObject(configb)...)
			repSpecb.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))
			forExpr := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true)
			innerTokens := hcl.TokensFromExpr(forExpr)
			innerTokens = append(innerTokens, hcl.TokensArraySingle(repSpecb)...)
			tokens := hcl.TokensFuncFlatten(innerTokens)
			resourceb.RemoveBlock(dSpec.block)
			resourceb.SetAttributeRaw(nRepSpecs, tokens)
			return dSpec, nil
		}
		numShardsAttr := dSpec.content.Body().GetAttribute(nNumShards)
		dSpec.content.Body().RemoveAttribute(nNumShards)
		if err := convertConfig(dSpec.content.Body(), diskSizeGB); err != nil {
			return dynamicBlock{}, err
		}
		var tokens hclwrite.Tokens
		if numShardsAttr != nil {
			numShardsExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
			outerFor := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), false)
			innerFor := buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false)
			forExpr := fmt.Sprintf("%s [\n    %s ", outerFor, innerFor)
			tokens = hcl.TokensFromExpr(forExpr)
			tokens = append(tokens, hcl.TokensObject(dSpec.content.Body())...)
			tokens = append(tokens, hcl.TokensFromExpr("\n  ]\n]")...)
			tokens = hcl.TokensFuncFlatten(tokens)
		} else {
			forExpr := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), false)
			tokens = hcl.TokensFromExpr(forExpr)
			tokens = append(tokens, hcl.TokensObject(dSpec.content.Body())...)
			tokens = hcl.EncloseBracketsNewLines(tokens)
		}
		resourceb.RemoveBlock(dSpec.block)
		resourceb.SetAttributeRaw(nRepSpecs, tokens)
		return dSpec, nil
	}
	return dynamicBlock{}, nil
}

func addAttributesWithTransform(targetBody *hclwrite.Body, sourceAttrs map[string]*hclwrite.Attribute,
	configBlockName string) {
	transform1 := func(expr string) string {
		return replaceDynamicBlockReferences(expr, configBlockName, nRegion)
	}
	transform2 := func(expr string) string {
		return replaceDynamicBlockReferences(expr, nRepSpecs, nSpec)
	}
	transformAttributesSorted(targetBody, sourceAttrs, transform1, transform2)
}

func convertConfig(repSpecs *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
	dConfig, err := getDynamicBlock(repSpecs, nConfig)
	if err != nil {
		return err
	}
	if dConfig.IsPresent() {
		blockName := getResourceName(dConfig.block)
		transform := func(expr string) string {
			return replaceDynamicBlockReferences(expr, blockName, nRegion)
		}
		transformAttributesSorted(dConfig.content.Body(), dConfig.content.Body().Attributes(), transform)
		for _, block := range dConfig.content.Body().Blocks() {
			transformAttributesSorted(block.Body(), block.Body().Attributes(), transform)
		}
		processAllSpecs(dConfig.content.Body(), diskSizeGB)
		forExpr := buildForExpr(nRegion, hcl.GetAttrExpr(dConfig.forEach), false)
		tokens := hcl.TokensFromExpr(forExpr)
		tokens = append(tokens, hcl.TokensObject(dConfig.content.Body())...)
		tokens = hcl.EncloseBracketsNewLines(tokens)
		repSpecs.RemoveBlock(dConfig.block)
		repSpecs.SetAttributeRaw(nConfig, tokens)
		return nil
	}
	var configs []*hclwrite.Body
	for _, block := range collectBlocks(repSpecs, nConfig) {
		blockb := block.Body()
		processAllSpecs(blockb, diskSizeGB)
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
	expectedBlocks := []string{nRepSpecs, nTags, nLabels, nAdvConfig, nBiConnector, nPinnedFCV, nTimeouts}
	for name := range resourceb.Attributes() {
		if slices.Contains(expectedBlocks, name) {
			return true
		}
	}
	return false
}

// copyAttributesSorted copies attributes from source to target in sorted order for deterministic output
func copyAttributesSorted(targetBody *hclwrite.Body, sourceAttrs map[string]*hclwrite.Attribute) {
	var names []string
	for name := range sourceAttrs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		attr := sourceAttrs[name]
		targetBody.SetAttributeRaw(name, hcl.TokensFromExpr(hcl.GetAttrExpr(attr)))
	}
}

// transformAttributesSorted transforms and copies attributes in sorted order
func transformAttributesSorted(targetBody *hclwrite.Body, sourceAttrs map[string]*hclwrite.Attribute,
	transforms ...func(string) string) {
	var names []string
	for name := range sourceAttrs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		attr := sourceAttrs[name]
		expr := hcl.GetAttrExpr(attr)
		// Apply all transformations
		for _, transform := range transforms {
			expr = transform(expr)
		}
		targetBody.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
}

// processAllSpecs processes all spec blocks (electable, read_only, analytics) and auto_scaling blocks
func processAllSpecs(body *hclwrite.Body, diskSizeGB hclwrite.Tokens) {
	fillSpecOpt(body, nElectableSpecs, diskSizeGB)
	fillSpecOpt(body, nReadOnlySpecs, diskSizeGB)
	fillSpecOpt(body, nAnalyticsSpecs, diskSizeGB)
	fillSpecOpt(body, nAutoScaling, nil)
	fillSpecOpt(body, nAnalyticsAutoScaling, nil)
}

// fillSpecOpt converts a spec block to an attribute with object value and optionally adds disk_size_gb
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
