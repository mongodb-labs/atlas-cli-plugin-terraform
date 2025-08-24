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
		resourceb.RemoveBlock(d.block)
		resourceb.SetAttributeRaw(nRepSpecs, d.tokens)
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
	if err != nil || !dSpec.IsPresent() {
		return dynamicBlock{}, err
	}
	transformDynamicBlockReferences(dSpec.content.Body(), nRepSpecs, nSpec)
	dConfig, err := convertConfigsWithDynamicBlock(dSpec.content.Body(), diskSizeGB)
	if err != nil {
		return dynamicBlock{}, err
	}
	forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
	forSpec = append(forSpec, dConfig.tokens...)
	tokens := hcl.TokensFuncFlatten(forSpec)
	dSpec.tokens = tokens
	return dSpec, nil
}

func convertConfigsWithDynamicBlock(specbSrc *hclwrite.Body, diskSizeGB hclwrite.Tokens) (dynamicBlock, error) {
	d, err := getDynamicBlock(specbSrc, nConfig)
	if err != nil {
		return dynamicBlock{}, err
	}
	if !d.IsPresent() {
		d, err = getDynamicBlock(specbSrc, nConfigSrc)
		if err != nil {
			return dynamicBlock{}, err
		}
	}
	if !d.IsPresent() {
		// No dynamic config block, handle num_shards if present
		numShardsAttr := specbSrc.GetAttribute(nNumShards)
		if numShardsAttr == nil {
			return dynamicBlock{}, fmt.Errorf("%s: %s not found", errRepSpecs, nNumShards)
		}
		specbSrc.RemoveAttribute(nNumShards)
		if errConv := convertConfig(specbSrc, diskSizeGB); errConv != nil {
			return dynamicBlock{}, errConv
		}
		numShardsExpr := hcl.GetAttrExpr(numShardsAttr)
		numShardsExpr = replaceDynamicBlockReferences(numShardsExpr, nRepSpecs, nSpec)
		tokens := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false))
		tokens = append(tokens, hcl.TokensObject(specbSrc)...)
		tokens = hcl.EncloseBracketsNewLines(tokens)
		return dynamicBlock{tokens: tokens}, nil
	}

	// Dynamic config block found
	repSpec := hclwrite.NewEmptyFile()
	repSpecb := repSpec.Body()
	if zoneNameAttr := specbSrc.GetAttribute(nZoneName); zoneNameAttr != nil {
		zoneNameExpr := hcl.GetAttrExpr(zoneNameAttr)
		zoneNameExpr = replaceDynamicBlockReferences(zoneNameExpr, nRepSpecs, nSpec)
		repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
	}

	configForEach := fmt.Sprintf("%s.%s", nSpec, nConfig)

	regionConfig, err := getDynamicRegionConfig(d, configForEach, diskSizeGB)
	if err != nil {
		return dynamicBlock{}, err
	}
	repSpecb.SetAttributeRaw(nConfig, regionConfig)

	// Handle num_shards
	numShardsAttr := specbSrc.GetAttribute(nNumShards)
	if numShardsAttr != nil {
		numShardsExpr := hcl.GetAttrExpr(numShardsAttr)
		numShardsExpr = replaceDynamicBlockReferences(numShardsExpr, nRepSpecs, nSpec)
		tokens := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false))
		tokens = append(tokens, hcl.TokensObject(repSpecb)...)
		return dynamicBlock{tokens: hcl.EncloseBracketsNewLines(tokens)}, nil
	}
	return dynamicBlock{tokens: hcl.TokensArraySingle(repSpecb)}, nil
}

// getDynamicRegionConfig builds the region config array for a dynamic config block
func getDynamicRegionConfig(d dynamicBlock, configForEach string, diskSizeGB hclwrite.Tokens) (hclwrite.Tokens, error) {
	configBlockName := getResourceName(d.block)
	transformDynamicBlockReferences(d.content.Body(), configBlockName, nRegion)
	for _, block := range d.content.Body().Blocks() {
		transformDynamicBlockReferences(block.Body(), configBlockName, nRegion)
	}
	// Additional transformation for nested spec references
	for name, attr := range d.content.Body().Attributes() {
		expr := replaceDynamicBlockReferences(hcl.GetAttrExpr(attr), nRepSpecs, nSpec)
		d.content.Body().SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
	for _, block := range d.content.Body().Blocks() {
		for name, attr := range block.Body().Attributes() {
			expr := replaceDynamicBlockReferences(hcl.GetAttrExpr(attr), nRepSpecs, nSpec)
			block.Body().SetAttributeRaw(name, hcl.TokensFromExpr(expr))
		}
	}

	regionConfigFile := hclwrite.NewEmptyFile()
	regionConfigBody := regionConfigFile.Body()
	copyAttributesSorted(regionConfigBody, d.content.Body().Attributes())
	for _, block := range d.content.Body().Blocks() {
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

	regionForExpr := buildForExpr(nRegion, configForEach, false)
	regionTokens := hcl.TokensFromExpr(regionForExpr)
	regionTokens = append(regionTokens, hcl.TokensObject(regionConfigBody)...)
	return hcl.EncloseBracketsNewLines(regionTokens), nil
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
