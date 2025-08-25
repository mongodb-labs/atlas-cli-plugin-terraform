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
func processNumShards(shardsAttr *hclwrite.Attribute, processedBody *hclwrite.Body) hclwrite.Tokens {
	if shardsAttr == nil {
		return hcl.TokensArraySingle(processedBody) // Default 1 if no num_shards specified
	}
	if shardsVal, err := hcl.GetAttrInt(shardsAttr, errNumShards); err == nil {
		var bodies []*hclwrite.Body
		for range shardsVal {
			bodies = append(bodies, processedBody)
		}
		return hcl.TokensArray(bodies)
	}
	shardsExpr := hcl.GetAttrExpr(shardsAttr)
	tokens := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", shardsExpr), false))
	tokens = append(tokens, hcl.TokensObject(processedBody)...)
	return hcl.EncloseBracketsNewLines(tokens)
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

func checkDynamicBlock(body *hclwrite.Body) error {
	dynamicBlockAllowList := []string{nTags, nLabels, nRepSpecs}
	for _, block := range body.Blocks() {
		name := getResourceName(block)
		if block.Type() != nDynamic || slices.Contains(dynamicBlockAllowList, name) {
			continue
		}
		return fmt.Errorf("dynamic blocks are not supported for %s", name)
	}
	return nil
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

// transformReference changes value references,
// e.g. regions_config.value.electable_nodes to region.electable_nodes
func transformReference(expr, blockName, varName string) string {
	return strings.ReplaceAll(expr,
		fmt.Sprintf("%s.%s.", blockName, nValue),
		fmt.Sprintf("%s.", varName))
}

// transformReferences transforms all attribute references in a body from dynamic block format
func transformReferences(body *hclwrite.Body, blockName, varName string) {
	for name, attr := range body.Attributes() {
		expr := transformReference(hcl.GetAttrExpr(attr), blockName, varName)
		body.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
	for _, block := range body.Blocks() {
		transformReferences(block.Body(), blockName, varName)
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

// buildForExpr builds a for expression with the given variable and collection
func buildForExpr(varName, collection string, trailingSpace bool) string {
	expr := fmt.Sprintf("for %s in %s :", varName, collection)
	if trailingSpace {
		expr += " "
	}
	return expr
}

func fillTagsLabelsOpt(resourceb *hclwrite.Body, name string) error {
	tokensDynamic, err := extractTagsLabelsDynamicBlock(resourceb, name)
	if err != nil {
		return err
	}
	tokensIndividual, err := extractTagsLabelsIndividual(resourceb, name)
	if err != nil {
		return err
	}
	if tokensDynamic != nil && tokensIndividual != nil {
		resourceb.SetAttributeRaw(name, hcl.TokensFuncMerge(tokensDynamic, tokensIndividual))
		return nil
	}
	if tokensDynamic != nil {
		resourceb.SetAttributeRaw(name, tokensDynamic)
	}
	if tokensIndividual != nil {
		resourceb.SetAttributeRaw(name, tokensIndividual)
	}
	return nil
}

func extractTagsLabelsDynamicBlock(resourceb *hclwrite.Body, name string) (hclwrite.Tokens, error) {
	d, err := getDynamicBlock(resourceb, name)
	if err != nil || !d.IsPresent() {
		return nil, err
	}
	key := d.content.Body().GetAttribute(nKey)
	value := d.content.Body().GetAttribute(nValue)
	if key == nil || value == nil {
		return nil, fmt.Errorf("dynamic block %s: %s or %s not found", name, nKey, nValue)
	}
	keyExpr := replaceDynamicBlockExpr(key, name, nKey)
	valueExpr := replaceDynamicBlockExpr(value, name, nValue)
	collectionExpr := hcl.GetAttrExpr(d.forEach)
	forExpr := fmt.Sprintf("for key, value in %s : %s => %s", collectionExpr, keyExpr, valueExpr)
	tokens := hcl.EncloseBraces(hcl.EncloseNewLines(hcl.TokensFromExpr(forExpr)), false)
	if keyExpr == nKey && valueExpr == nValue { // expression can be simplified and use for_each expression
		tokens = hcl.TokensFromExpr(collectionExpr)
	}
	resourceb.RemoveBlock(d.block)
	return tokens, nil
}

func extractTagsLabelsIndividual(resourceb *hclwrite.Body, name string) (hclwrite.Tokens, error) {
	var (
		file  = hclwrite.NewEmptyFile()
		fileb = file.Body()
		found = false
	)
	for {
		block := resourceb.FirstMatchingBlock(name, nil)
		if block == nil {
			break
		}
		key := block.Body().GetAttribute(nKey)
		value := block.Body().GetAttribute(nValue)
		if key == nil || value == nil {
			return nil, fmt.Errorf("%s: %s or %s not found", name, nKey, nValue)
		}
		setKeyValue(fileb, key, value)
		resourceb.RemoveBlock(block)
		found = true
	}
	if !found {
		return nil, nil
	}
	return hcl.TokensObject(fileb), nil
}
