package convert

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
)

// hasVariableNumShards checks if any block has a variable (non-literal) num_shards attribute
func hasVariableNumShards(blocks []*hclwrite.Block) bool {
	for _, block := range blocks {
		if shardsAttr := block.Body().GetAttribute(nNumShards); shardsAttr != nil {
			if _, err := hcl.GetAttrInt(shardsAttr, errNumShards); err != nil {
				return true
			}
		}
	}
	return false
}

// processNumShards handles num_shards for a block, returning tokens for the expanded specs.
// processedBody is the body with num_shards removed and other processing done.
func processNumShards(shardsAttr *hclwrite.Attribute, processedBody *hclwrite.Body) (hclwrite.Tokens, error) {
	if shardsAttr == nil {
		return hcl.TokensArraySingle(processedBody), nil // Default 1 if no num_shards specified
	}
	if shardsVal, err := hcl.GetAttrInt(shardsAttr, errNumShards); err == nil {
		var bodies []*hclwrite.Body
		for range shardsVal {
			bodies = append(bodies, processedBody)
		}
		return hcl.TokensArray(bodies), nil
	}
	shardsExpr := hcl.GetAttrExpr(shardsAttr)
	tokens := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", shardsExpr)))
	tokens = append(tokens, hcl.TokensObject(processedBody)...)
	return hcl.EncloseBracketsNewLines(tokens), nil
}

type dynamicBlock struct {
	block   *hclwrite.Block
	forEach *hclwrite.Attribute
	content *hclwrite.Block
	tokens  hclwrite.Tokens
}

func (d dynamicBlock) IsPresent() bool {
	return d.block != nil
}

// getDynamicBlock finds and returns a dynamic block with the given name from the body
func getDynamicBlock(body *hclwrite.Body, name string) (dynamicBlock, error) {
	for _, block := range body.Blocks() {
		if block.Type() != nDynamic || name != getResourceName(block) {
			continue
		}
		blockb := block.Body()
		forEach := blockb.GetAttribute(nForEach)
		if forEach == nil {
			return dynamicBlock{}, fmt.Errorf("dynamic block %s: attribute %s not found", name, nForEach)
		}
		content := blockb.FirstMatchingBlock(nContent, nil)
		if content == nil {
			return dynamicBlock{}, fmt.Errorf("dynamic block %s: block %s not found", name, nContent)
		}
		return dynamicBlock{forEach: forEach, block: block, content: content}, nil
	}
	return dynamicBlock{}, nil
}

// getResourceName returns the first label of a block, if it exists.
// e.g. in resource "mongodbatlas_cluster" "mycluster", the first label is "mongodbatlas_cluster".
func getResourceName(resource *hclwrite.Block) string {
	labels := resource.Labels()
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}

// replaceDynamicBlockReferences changes value references,
// e.g. regions_config.value.electable_nodes to region.electable_nodes
func replaceDynamicBlockReferences(expr, blockName, varName string) string {
	return strings.ReplaceAll(expr,
		fmt.Sprintf("%s.%s.", blockName, nValue),
		fmt.Sprintf("%s.", varName))
}

// transformDynamicBlockReferences transforms all attribute references in a body from dynamic block format
func transformDynamicBlockReferences(configSrcb *hclwrite.Body, blockName, varName string) {
	for name, attr := range configSrcb.Attributes() {
		expr := replaceDynamicBlockReferences(hcl.GetAttrExpr(attr), blockName, varName)
		configSrcb.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
}

// transformDynamicBlockReferencesRecursive transforms attributes and nested blocks recursively
// replacing references from dynamic block format, e.g. regions_config.value.* to region.*
func transformDynamicBlockReferencesRecursive(body *hclwrite.Body, blockName, varName string) {
	transform := func(expr string) string {
		return replaceDynamicBlockReferences(expr, blockName, varName)
	}
	transformAttributesSorted(body, body.Attributes(), transform)
	for _, block := range body.Blocks() {
		transformDynamicBlockReferencesRecursive(block.Body(), blockName, varName)
	}
}

// collectBlocks removes and returns all blocks of the given name from body in order of appearance.
func collectBlocks(body *hclwrite.Body, name string) []*hclwrite.Block {
	var blocks []*hclwrite.Block
	for {
		block := body.FirstMatchingBlock(name, nil)
		if block == nil {
			break
		}
		body.RemoveBlock(block)
		blocks = append(blocks, block)
	}
	return blocks
}

// fillBlockOpt converts a block to an attribute with object value
func fillBlockOpt(resourceb *hclwrite.Body, name string) {
	block := resourceb.FirstMatchingBlock(name, nil)
	if block == nil {
		return
	}
	resourceb.RemoveBlock(block)
	resourceb.SetAttributeRaw(name, hcl.TokensObject(block.Body()))
}

// fillAdvConfigOpt fills the advanced_configuration attribute, removing deprecated attributes
func fillAdvConfigOpt(resourceb *hclwrite.Body) {
	block := resourceb.FirstMatchingBlock(nAdvConfig, nil)
	if block == nil {
		return
	}
	blockBody := block.Body()

	// Remove deprecated attributes from advanced_configuration
	blockBody.RemoveAttribute(nFailIndexKeyTooLong)
	blockBody.RemoveAttribute(nDefaultReadConcern)

	fillBlockOpt(resourceb, nAdvConfig)
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

// buildForExpr builds a for expression with the given variable and collection
func buildForExpr(varName, collection string) string {
	return fmt.Sprintf("for %s in %s :", varName, collection)
}
