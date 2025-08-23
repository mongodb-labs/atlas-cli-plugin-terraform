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
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs)
	if err != nil {
		return err
	}
	if dSpec.IsPresent() {
		return convertDynamicRepSpecs(resourceb, dSpec, diskSizeGB)
	}
	repSpecBlocks := collectBlocks(resourceb, nRepSpecs)
	if len(repSpecBlocks) == 0 {
		return fmt.Errorf("must have at least one replication_specs")
	}
	var tokens hclwrite.Tokens
	if hasVariableNumShards(repSpecBlocks) {
		tokens, err = processVariableNumShards(repSpecBlocks, diskSizeGB)
	} else {
		tokens, err = processStaticNumShards(repSpecBlocks, diskSizeGB)
	}
	if err != nil {
		return err
	}
	resourceb.SetAttributeRaw(nRepSpecs, tokens)
	return nil
}

func convertDynamicRepSpecs(resourceb *hclwrite.Body, dSpec dynamicBlock, diskSizeGB hclwrite.Tokens) error {
	transformDynamicBlockReferences(dSpec.content.Body(), nRepSpecs, nSpec)
	dConfig, err := findDynamicConfigBlock(dSpec.content.Body())
	if err != nil {
		return err
	}
	if dConfig.IsPresent() {
		return convertDynamicRepSpecsWithDynamicConfig(resourceb, dSpec, dConfig, diskSizeGB)
	}
	tokens, err := processDynamicRepSpecsWithoutConfig(dSpec, diskSizeGB)
	if err != nil {
		return err
	}
	resourceb.RemoveBlock(dSpec.block)
	resourceb.SetAttributeRaw(nRepSpecs, tokens)
	return nil
}

func processVariableNumShards(repSpecBlocks []*hclwrite.Block, diskSizeGB hclwrite.Tokens) (hclwrite.Tokens, error) {
	var concatParts []hclwrite.Tokens
	for _, block := range repSpecBlocks {
		blockb := block.Body()
		numShardsAttr := blockb.GetAttribute(nNumShards)
		blockb.RemoveAttribute(nNumShards)
		if err := convertConfig(blockb, diskSizeGB); err != nil {
			return nil, err
		}
		tokens, err := processNumShards(numShardsAttr, blockb)
		if err != nil {
			return nil, err
		}
		concatParts = append(concatParts, tokens)
	}
	return hcl.TokensFuncConcat(concatParts...), nil
}

func processStaticNumShards(repSpecBlocks []*hclwrite.Block, diskSizeGB hclwrite.Tokens) (hclwrite.Tokens, error) {
	var repSpecs []*hclwrite.Body
	for _, block := range repSpecBlocks {
		blockb := block.Body()
		numShardsAttr := blockb.GetAttribute(nNumShards)
		blockb.RemoveAttribute(nNumShards)
		if err := convertConfig(blockb, diskSizeGB); err != nil {
			return nil, err
		}
		if numShardsAttr != nil {
			numShardsVal, _ := hcl.GetAttrInt(numShardsAttr, errNumShards)
			for range numShardsVal {
				repSpecs = append(repSpecs, blockb)
			}
		} else {
			repSpecs = append(repSpecs, blockb)
		}
	}
	return hcl.TokensArray(repSpecs), nil
}

func findDynamicConfigBlock(body *hclwrite.Body) (dynamicBlock, error) {
	dConfig, err := getDynamicBlock(body, nConfig)
	if err != nil {
		return dynamicBlock{}, err
	}
	if !dConfig.IsPresent() {
		dConfig, err = getDynamicBlock(body, nConfigSrc)
		if err != nil {
			return dynamicBlock{}, err
		}
	}
	return dConfig, nil
}

func processDynamicRepSpecsWithoutConfig(dSpec dynamicBlock, diskSizeGB hclwrite.Tokens) (hclwrite.Tokens, error) {
	numShardsAttr := dSpec.content.Body().GetAttribute(nNumShards)
	dSpec.content.Body().RemoveAttribute(nNumShards)
	if err := convertConfig(dSpec.content.Body(), diskSizeGB); err != nil {
		return nil, err
	}
	if numShardsAttr != nil {
		return buildDynamicRepSpecsWithShards(dSpec, numShardsAttr)
	}
	return buildSimpleDynamicRepSpecs(dSpec)
}

func buildDynamicRepSpecsWithShards(dSpec dynamicBlock, numShardsAttr *hclwrite.Attribute) (hclwrite.Tokens, error) {
	numShardsExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
	outerFor := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), false)
	innerFor := buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false)
	forExpr := fmt.Sprintf("%s [\n    %s ", outerFor, innerFor)
	tokens := hcl.TokensFromExpr(forExpr)
	tokens = append(tokens, hcl.TokensObject(dSpec.content.Body())...)
	tokens = append(tokens, hcl.TokensFromExpr("\n  ]\n]")...)
	return hcl.TokensFuncFlatten(tokens), nil
}

func buildSimpleDynamicRepSpecs(dSpec dynamicBlock) (hclwrite.Tokens, error) {
	forExpr := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), false)
	tokens := hcl.TokensFromExpr(forExpr)
	tokens = append(tokens, hcl.TokensObject(dSpec.content.Body())...)
	return hcl.EncloseBracketsNewLines(tokens), nil
}

func buildDynamicRepSpecsWithNumShards(dSpec, dConfig dynamicBlock, diskSizeGB hclwrite.Tokens,
	configBlockName string, numShardsAttr *hclwrite.Attribute) (hclwrite.Tokens, error) {
	numShardsExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
	transformDynamicBlockReferencesRecursive(dConfig.content.Body(), configBlockName, nRegion)
	transform := func(expr string) string {
		return replaceDynamicBlockReferences(expr, nRepSpecs, nSpec)
	}
	transformAttributesSorted(dConfig.content.Body(), dConfig.content.Body().Attributes(), transform)
	for _, block := range dConfig.content.Body().Blocks() {
		transformAttributesSorted(block.Body(), block.Body().Attributes(), transform)
	}
	regionConfigBody := buildRegionConfigBody(dConfig, diskSizeGB)

	configForEach := fmt.Sprintf("%s.%s", nSpec, nConfig)
	regionForExpr := buildForExpr(nRegion, configForEach, false)
	regionTokens := hcl.TokensFromExpr(regionForExpr)
	regionTokens = append(regionTokens, hcl.TokensObject(regionConfigBody)...)

	repSpecBody := buildRepSpecBody(dSpec, regionTokens)

	// Inline buildInnerForExpr
	innerForExpr := buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false)
	innerTokens := hcl.TokensFromExpr(innerForExpr)
	innerTokens = append(innerTokens, hcl.TokensObject(repSpecBody)...)

	// Inline buildOuterForExpr
	outerForExpr := buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true)
	outerTokens := hcl.TokensFromExpr(outerForExpr)
	outerTokens = append(outerTokens, hcl.EncloseBracketsNewLines(innerTokens)...)

	return hcl.TokensFuncFlatten(outerTokens), nil
}

func buildRegionConfigBody(dConfig dynamicBlock, diskSizeGB hclwrite.Tokens) *hclwrite.Body {
	regionConfigFile := hclwrite.NewEmptyFile()
	regionConfigBody := regionConfigFile.Body()
	copyAttributesSorted(regionConfigBody, dConfig.content.Body().Attributes())
	processRegionConfigBlocks(regionConfigBody, dConfig.content.Body().Blocks(), diskSizeGB)
	return regionConfigBody
}

func buildRepSpecBody(dSpec dynamicBlock, regionTokens hclwrite.Tokens) *hclwrite.Body {
	repSpecFile := hclwrite.NewEmptyFile()
	repSpecBody := repSpecFile.Body()
	if zoneNameAttr := dSpec.content.Body().GetAttribute(nZoneName); zoneNameAttr != nil {
		zoneNameExpr := replaceDynamicBlockReferences(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
		repSpecBody.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
	}
	repSpecBody.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))
	return repSpecBody
}

func processRegionConfigBlocks(targetBody *hclwrite.Body, blocks []*hclwrite.Block, diskSizeGB hclwrite.Tokens) {
	for _, block := range blocks {
		blockType := block.Type()
		blockFile := hclwrite.NewEmptyFile()
		blockBody := blockFile.Body()
		copyAttributesSorted(blockBody, block.Body().Attributes())
		if diskSizeGB != nil && (blockType == nElectableSpecs ||
			blockType == nReadOnlySpecs || blockType == nAnalyticsSpecs) {
			blockBody.SetAttributeRaw(nDiskSizeGB, diskSizeGB)
		}
		targetBody.SetAttributeRaw(blockType, hcl.TokensObject(blockBody))
	}
}

func convertDynamicRepSpecsWithDynamicConfig(resourceb *hclwrite.Body, dSpec, dConfig dynamicBlock,
	diskSizeGB hclwrite.Tokens) error {
	configBlockName := getResourceName(dConfig.block)
	numShardsAttr := dSpec.content.Body().GetAttribute(nNumShards)
	if numShardsAttr != nil {
		tokens, err := buildDynamicRepSpecsWithNumShards(dSpec, dConfig, diskSizeGB, configBlockName, numShardsAttr)
		if err != nil {
			return err
		}
		resourceb.RemoveBlock(dSpec.block)
		resourceb.SetAttributeRaw(nRepSpecs, tokens)
		return nil
	}
	return convertDynamicRepSpecsWithoutNumShards(resourceb, dSpec, dConfig, diskSizeGB, configBlockName)
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

func convertDynamicRepSpecsWithoutNumShards(resourceb *hclwrite.Body, dSpec, dConfig dynamicBlock,
	diskSizeGB hclwrite.Tokens, configBlockName string) error {
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
	return nil
}

func convertConfig(repSpecs *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
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
		return convertDynamicConfig(repSpecs, dConfig, diskSizeGB)
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

func convertDynamicConfig(repSpecs *hclwrite.Body, dConfig dynamicBlock, diskSizeGB hclwrite.Tokens) error {
	blockName := getResourceName(dConfig.block)
	transformDynamicBlockReferencesRecursive(dConfig.content.Body(), blockName, nRegion)
	processAllSpecs(dConfig.content.Body(), diskSizeGB)
	forExpr := buildForExpr(nRegion, hcl.GetAttrExpr(dConfig.forEach), false)
	tokens := hcl.TokensFromExpr(forExpr)
	tokens = append(tokens, hcl.TokensObject(dConfig.content.Body())...)
	tokens = hcl.EncloseBracketsNewLines(tokens)
	repSpecs.RemoveBlock(dConfig.block)
	repSpecs.SetAttributeRaw(nConfig, tokens)
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
