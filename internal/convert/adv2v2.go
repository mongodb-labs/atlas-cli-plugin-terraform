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
		dConfig, err := convertConfigsWithDynamicBlock(blockb, diskSizeGB, false)
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

func convertRepSpecsWithDynamicBlock(resourceb *hclwrite.Body, diskSizeGB hclwrite.Tokens) (dynamicBlock, error) {
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs, true)
	if err != nil || !dSpec.IsPresent() {
		return dynamicBlock{}, err
	}
	transformReferences(dSpec.content.Body(), nRepSpecs, nSpec)
	dConfig, err := convertConfigsWithDynamicBlock(dSpec.content.Body(), diskSizeGB, true)
	if err != nil {
		return dynamicBlock{}, err
	}

	// Check if we have a dynamic region_configs block that was successfully processed
	if dConfig.tokens != nil {
		forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
		dSpec.tokens = hcl.TokensFuncFlatten(append(forSpec, dConfig.tokens...))
		return dSpec, nil
	}

	// Handle static region_configs blocks inside dynamic replication_specs
	specBody := dSpec.content.Body()

	// Collect static region_configs blocks
	staticConfigs := collectBlocks(specBody, nConfig)
	if len(staticConfigs) == 0 {
		// No static blocks found, this might be an error case
		// Check if there's also no dynamic block (which would have been handled above)
		hasDynamicBlock := false
		for _, block := range specBody.Blocks() {
			if block.Type() == nDynamic && getResourceName(block) == nConfig {
				hasDynamicBlock = true
				break
			}
		}
		if !hasDynamicBlock {
			return dynamicBlock{}, fmt.Errorf("replication_specs must have at least one region_configs")
		}
		// There's a dynamic block but convertConfigsWithDynamicBlock returned empty
		// This shouldn't happen, but return the error from that function
		return dynamicBlock{}, nil
	}

	repSpecb := hclwrite.NewEmptyFile().Body()

	// Handle zone_name attribute
	if zoneNameAttr := specBody.GetAttribute(nZoneName); zoneNameAttr != nil {
		zoneNameExpr := transformReference(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
		repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
	}

	// Process static region_configs blocks
	var configs []*hclwrite.Body
	for _, configBlock := range staticConfigs {
		configBlockb := configBlock.Body()
		// Create a new body with sorted attributes
		newConfigBody := hclwrite.NewEmptyFile().Body()

		// Copy attributes in the expected order
		attrs := configBlockb.Attributes()
		// Priority, provider_name, region_name should come first
		if priority := attrs["priority"]; priority != nil {
			newConfigBody.SetAttributeRaw("priority", priority.Expr().BuildTokens(nil))
		}
		if provider := attrs["provider_name"]; provider != nil {
			newConfigBody.SetAttributeRaw("provider_name", provider.Expr().BuildTokens(nil))
		}
		if region := attrs["region_name"]; region != nil {
			newConfigBody.SetAttributeRaw("region_name", region.Expr().BuildTokens(nil))
		}

		// Process spec blocks and convert them to attributes
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

		configs = append(configs, newConfigBody)
	}

	repSpecb.SetAttributeRaw(nConfig, hcl.TokensArray(configs))

	// Handle num_shards attribute
	if numShardsAttr := specBody.GetAttribute(nNumShards); numShardsAttr != nil {
		numShardsExpr := transformReference(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
		forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
		innerFor := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false))
		innerFor = append(innerFor, hcl.TokensObject(repSpecb)...)
		dSpec.tokens = hcl.TokensFuncFlatten(append(forSpec, hcl.EncloseBracketsNewLines(innerFor)...))
	} else {
		forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
		dSpec.tokens = hcl.TokensFuncFlatten(append(forSpec, hcl.TokensArraySingle(repSpecb)...))
	}

	return dSpec, nil
}

func convertConfigsWithDynamicBlock(specbSrc *hclwrite.Body, diskSizeGB hclwrite.Tokens,
	insideDynamicRepSpec bool) (dynamicBlock, error) {
	d, err := getDynamicBlock(specbSrc, nConfig, true)
	if err != nil || !d.IsPresent() {
		return dynamicBlock{}, err
	}
	configBody := d.content.Body()
	transformReferences(configBody, getResourceName(d.block), nRegion)
	regionConfigBody := hclwrite.NewEmptyFile().Body()
	copyAttributesSorted(regionConfigBody, configBody.Attributes())
	for _, block := range configBody.Blocks() {
		blockType := block.Type()
		blockBody := hclwrite.NewEmptyFile().Body()
		copyAttributesSorted(blockBody, block.Body().Attributes())
		if diskSizeGB != nil &&
			(blockType == nElectableSpecs || blockType == nReadOnlySpecs || blockType == nAnalyticsSpecs) {
			blockBody.SetAttributeRaw(nDiskSizeGB, diskSizeGB)
		}
		regionConfigBody.SetAttributeRaw(blockType, hcl.TokensObject(blockBody))
	}
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
	if zoneNameAttr := specbSrc.GetAttribute(nZoneName); zoneNameAttr != nil {
		zoneNameExpr := transformReference(hcl.GetAttrExpr(zoneNameAttr), nRepSpecs, nSpec)
		repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneNameExpr))
	}
	repSpecb.SetAttributeRaw(nConfig, hcl.EncloseBracketsNewLines(regionTokens))
	if numShardsAttr := specbSrc.GetAttribute(nNumShards); numShardsAttr != nil {
		numShardsExpr := transformReference(hcl.GetAttrExpr(numShardsAttr), nRepSpecs, nSpec)
		tokens := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", numShardsExpr), false))
		tokens = append(tokens, hcl.TokensObject(repSpecb)...)
		return dynamicBlock{tokens: hcl.EncloseBracketsNewLines(tokens)}, nil
	}
	return dynamicBlock{tokens: hcl.TokensArraySingle(repSpecb)}, nil
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
	fillSpecOpt(body, nElectableSpecs, diskSizeGB)
	fillSpecOpt(body, nReadOnlySpecs, diskSizeGB)
	fillSpecOpt(body, nAnalyticsSpecs, diskSizeGB)
	fillSpecOpt(body, nAutoScaling, nil)
	fillSpecOpt(body, nAnalyticsAutoScaling, nil)
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
