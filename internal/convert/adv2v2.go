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
		updated, err := processResource(block)
		if err != nil {
			return nil, err
		}
		if updated {
			addConversionComments(block, true)
		}
	}
	return parser.Bytes(), nil
}

func processResource(resource *hclwrite.Block) (bool, error) {
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
	if err := processRepSpecs(resourceb, diskSizeGB); err != nil {
		return false, err
	}
	if err := processCommonOptionalBlocks(resourceb); err != nil {
		return false, err
	}
	return true, nil
}

func processRepSpecs(resourceb *hclwrite.Body, diskSizeGB hclwrite.Tokens) error {
	d, err := processRepSpecsWithDynamicBlock(resourceb, diskSizeGB)
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
		dConfig, err := processConfigsWithDynamicBlock(blockb, diskSizeGB, false)
		if err != nil {
			return err
		}
		if dConfig.IsPresent() {
			blockb.RemoveBlock(dConfig.block)
			blockb.SetAttributeRaw(nConfig, dConfig.tokens)
		} else {
			var configs []*hclwrite.Body
			for _, configBlock := range collectBlocks(blockb, nConfig) {
				configBlockb := configBlock.Body()
				processAllSpecs(configBlockb, diskSizeGB)
				configs = append(configs, configBlockb)
			}
			if len(configs) == 0 {
				return fmt.Errorf("replication_specs must have at least one region_configs")
			}
			blockb.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
		}
		if hasVariableShards {
			resultTokens = append(resultTokens, processNumShardsWhenSomeIsVariable(shardsAttr, blockb))
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

func processRepSpecsWithDynamicBlock(resourceb *hclwrite.Body, diskSizeGB hclwrite.Tokens) (dynamicBlock, error) {
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs, true)
	if err != nil || !dSpec.IsPresent() {
		return dynamicBlock{}, err
	}
	transformReferences(dSpec.content.Body(), nRepSpecs, nSpec)
	dConfig, err := processConfigsWithDynamicBlock(dSpec.content.Body(), diskSizeGB, true)
	if err != nil {
		return dynamicBlock{}, err
	}
	if dConfig.tokens != nil {
		forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
		dSpec.tokens = hcl.TokensFuncFlatten(append(forSpec, dConfig.tokens...))
		return dSpec, nil
	}

	// Handle static region_configs blocks inside dynamic replication_specs
	specBody := dSpec.content.Body()
	staticConfigs := collectBlocks(specBody, nConfig)
	repSpecb := hclwrite.NewEmptyFile().Body()
	handleZoneName(repSpecb, specBody, nRepSpecs, nSpec)
	var configs []*hclwrite.Body
	for _, configBlock := range staticConfigs {
		configBlockb := configBlock.Body()
		newConfigBody := processConfigForDynamicBlock(configBlockb, diskSizeGB)
		configs = append(configs, newConfigBody)
	}
	repSpecb.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
	numShardsAttr := specBody.GetAttribute(nNumShards)
	forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
	numShardsTokens := buildNumShardsTokens(numShardsAttr, repSpecb, nRepSpecs, nSpec)
	dSpec.tokens = hcl.TokensFuncFlatten(append(forSpec, numShardsTokens...))
	return dSpec, nil
}

func processConfigsWithDynamicBlock(specbSrc *hclwrite.Body, diskSizeGB hclwrite.Tokens,
	insideDynamicRepSpec bool) (dynamicBlock, error) {
	d, err := getDynamicBlock(specbSrc, nConfig, true)
	if err != nil || !d.IsPresent() {
		return dynamicBlock{}, err
	}
	configBody := d.content.Body()
	transformReferences(configBody, getResourceName(d.block), nRegion)
	regionConfigBody := processConfigForDynamicBlock(configBody, diskSizeGB)
	forEach := hcl.GetAttrExpr(d.forEach)
	if insideDynamicRepSpec {
		forEach = fmt.Sprintf("%s.%s", nSpec, nConfig)
	}
	regionTokens := hcl.TokensFromExpr(buildForExpr(nRegion, forEach, false))
	regionTokens = append(regionTokens, hcl.TokensObject(regionConfigBody)...)
	if !insideDynamicRepSpec {
		d.tokens = hcl.EncloseBracketsNewLines(regionTokens)
		return d, nil
	}
	repSpecb := hclwrite.NewEmptyFile().Body()
	handleZoneName(repSpecb, specbSrc, nRepSpecs, nSpec)
	repSpecb.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))
	numShardsAttr := specbSrc.GetAttribute(nNumShards)
	tokens := buildNumShardsTokens(numShardsAttr, repSpecb, nRepSpecs, nSpec)
	return dynamicBlock{tokens: tokens}, nil
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

func copyAttributesSorted(targetBody *hclwrite.Body, sourceAttrs map[string]*hclwrite.Attribute) {
	var names []string
	for name := range sourceAttrs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		expr := hcl.GetAttrExpr(sourceAttrs[name])
		targetBody.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
}

func processAllSpecs(body *hclwrite.Body, diskSizeGB hclwrite.Tokens) {
	specsWithDisk := []string{nElectableSpecs, nReadOnlySpecs, nAnalyticsSpecs}
	for _, spec := range specsWithDisk {
		fillSpecOpt(body, spec, diskSizeGB)
	}
	specsWithoutDisk := []string{nAutoScaling, nAnalyticsAutoScaling}
	for _, spec := range specsWithoutDisk {
		fillSpecOpt(body, spec, nil)
	}
}

func processConfigForDynamicBlock(configBlockb *hclwrite.Body, diskSizeGB hclwrite.Tokens) *hclwrite.Body {
	newConfigBody := hclwrite.NewEmptyFile().Body()
	attrs := configBlockb.Attributes()
	orderedAttrs := []string{nPriority, nProviderName, nRegionName}
	for _, attrName := range orderedAttrs {
		if attr := attrs[attrName]; attr != nil {
			newConfigBody.SetAttributeRaw(attrName, attr.Expr().BuildTokens(nil))
		}
	}
	for _, block := range configBlockb.Blocks() {
		blockType := block.Type()
		blockBody := hclwrite.NewEmptyFile().Body()
		copyAttributesSorted(blockBody, block.Body().Attributes())
		if diskSizeGB != nil &&
			(blockType == nElectableSpecs || blockType == nReadOnlySpecs || blockType == nAnalyticsSpecs) {
			blockBody.SetAttributeRaw(nDiskSizeGB, diskSizeGB)
		}
		newConfigBody.SetAttributeRaw(blockType, hcl.TokensObject(blockBody))
	}
	return newConfigBody
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
