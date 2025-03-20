package convert

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/zclconf/go-cty/cty"
)

const (
	resourceType     = "resource"
	dataSourceType   = "data"
	cluster          = "mongodbatlas_cluster"
	advCluster       = "mongodbatlas_advanced_cluster"
	clusterPlural    = "mongodbatlas_clusters"
	advClusterPlural = "mongodbatlas_advanced_clusters"
	valClusterType   = "REPLICASET"
	valMaxPriority   = 7
	valMinPriority   = 0
	errFreeCluster   = "free cluster (because no " + nRepSpecs + ")"
	errRepSpecs      = "setting " + nRepSpecs
	errConfigs       = "setting " + nConfig
	errPriority      = "setting " + nPriority
	errNumShards     = "setting " + nNumShards

	commentGeneratedBy       = "Generated by atlas-cli-plugin-terraform."
	commentConfirmReferences = "Please review the changes and confirm that references to this resource are updated."
	commentMovedBlock        = "Moved blocks"
	commentRemovedOld        = "Note: Remember to remove or comment out the old cluster definitions."
	commentPriorityFor       = "Regions must be sorted by priority in descending order."
)

var (
	dynamicBlockAllowList = []string{nTags, nLabels, nConfigSrc, nRepSpecs}
)

type attrVals struct {
	req map[string]hclwrite.Tokens
	opt map[string]hclwrite.Tokens
}

// ClusterToAdvancedCluster transforms all mongodbatlas_cluster definitions in a
// Terraform configuration file into mongodbatlas_advanced_cluster schema 2.0.0.
// All other resources and data sources are left untouched.
// Note: hclwrite.Tokens are used instead of cty.Value so expressions with interpolations like var.region can be preserved.
// cty.Value only supports literal expressions.
func ClusterToAdvancedCluster(config []byte, includeMoved bool) ([]byte, error) {
	var moveLabels []string
	parser, err := hcl.GetParser(config)
	if err != nil {
		return nil, err
	}
	parserb := parser.Body()
	for _, block := range parserb.Blocks() {
		convertedResource, err := convertResource(block)
		if err != nil {
			return nil,
				err
		}
		if includeMoved && convertedResource {
			if moveLabel := getResourceLabel(block); moveLabel != "" {
				moveLabels = append(moveLabels, moveLabel)
			}
		}
		convertedDataSource := convertDataSource(block)
		if convertedResource || convertedDataSource {
			blockb := block.Body()
			blockb.AppendNewline()
			hcl.AppendComment(blockb, commentGeneratedBy)
			hcl.AppendComment(blockb, commentConfirmReferences)
		}
	}
	fillMovedBlocks(parserb, moveLabels)
	return parser.Bytes(), nil
}

func convertResource(block *hclwrite.Block) (bool, error) {
	if block.Type() != resourceType || getResourceName(block) != cluster {
		return false, nil
	}
	setResourceName(block, advCluster)
	blockb := block.Body()
	if errDyn := checkDynamicBlock(blockb); errDyn != nil {
		return false, errDyn
	}

	var err error
	if isFreeTierCluster(blockb) {
		err = fillFreeTierCluster(blockb)
	} else {
		err = fillCluster(blockb)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func isFreeTierCluster(resourceb *hclwrite.Body) bool {
	d, _ := getDynamicBlock(resourceb, nRepSpecs)
	return resourceb.FirstMatchingBlock(nRepSpecs, nil) == nil && !d.IsPresent()
}

func convertDataSource(block *hclwrite.Block) bool {
	if block.Type() != dataSourceType {
		return false
	}
	convertMap := map[string]string{
		cluster:       advCluster,
		clusterPlural: advClusterPlural,
	}
	if newName, found := convertMap[getResourceName(block)]; found {
		setResourceName(block, newName)
		block.Body().SetAttributeValue(nUseRepSpecsPerShard, cty.True)
		return true
	}
	return false
}

func fillMovedBlocks(body *hclwrite.Body, moveLabels []string) {
	if len(moveLabels) == 0 {
		return
	}
	body.AppendNewline()
	hcl.AppendComment(body, commentMovedBlock)
	hcl.AppendComment(body, commentRemovedOld)
	body.AppendNewline()
	for i, moveLabel := range moveLabels {
		block := body.AppendNewBlock(nMoved, nil)
		blockb := block.Body()
		blockb.SetAttributeRaw(nFrom, hcl.TokensFromExpr(fmt.Sprintf("%s.%s", cluster, moveLabel)))
		blockb.SetAttributeRaw(nTo, hcl.TokensFromExpr(fmt.Sprintf("%s.%s", advCluster, moveLabel)))
		if i < len(moveLabels)-1 {
			body.AppendNewline()
		}
	}
}

// fillFreeTierCluster is the entry point to convert clusters in free tier
func fillFreeTierCluster(resourceb *hclwrite.Body) error {
	resourceb.SetAttributeValue(nClusterType, cty.StringVal(valClusterType))
	config := hclwrite.NewEmptyFile()
	configb := config.Body()
	hcl.SetAttrInt(configb, nPriority, valMaxPriority)
	if err := hcl.MoveAttr(resourceb, configb, nRegionNameSrc, nRegionName, errFreeCluster); err != nil {
		return err
	}
	if err := hcl.MoveAttr(resourceb, configb, nProviderName, nProviderName, errFreeCluster); err != nil {
		return err
	}
	if err := hcl.MoveAttr(resourceb, configb, nBackingProviderName, nBackingProviderName, errFreeCluster); err != nil {
		return err
	}
	electableSpec := hclwrite.NewEmptyFile()
	if err := hcl.MoveAttr(resourceb, electableSpec.Body(), nInstanceSizeSrc, nInstanceSize, errFreeCluster); err != nil {
		return err
	}
	configb.SetAttributeRaw(nElectableSpecs, hcl.TokensObject(electableSpec.Body()))

	repSpecs := hclwrite.NewEmptyFile()
	repSpecs.Body().SetAttributeRaw(nConfig, hcl.TokensArraySingle(configb))
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArraySingle(repSpecs.Body()))
	return nil
}

// fillCluster is the entry point to convert clusters with replications_specs (all but free tier)
func fillCluster(resourceb *hclwrite.Body) error {
	root, errRoot := popRootAttrs(resourceb)
	if errRoot != nil {
		return errRoot
	}
	resourceb.RemoveAttribute(nNumShards) // num_shards in root is not relevant, only in replication_specs
	// ok to fail as cloud_backup is optional
	_ = hcl.MoveAttr(resourceb, resourceb, nCloudBackup, nBackupEnabled, errRepSpecs)
	if err := fillReplicationSpecs(resourceb, root); err != nil {
		return err
	}
	if err := fillTagsLabelsOpt(resourceb, nTags); err != nil {
		return err
	}
	if err := fillTagsLabelsOpt(resourceb, nLabels); err != nil {
		return err
	}
	fillBlockOpt(resourceb, nTimeouts)
	fillBlockOpt(resourceb, nAdvConf)
	fillBlockOpt(resourceb, nBiConnector)
	fillBlockOpt(resourceb, nPinnedFCV)
	return nil
}

func fillReplicationSpecs(resourceb *hclwrite.Body, root attrVals) error {
	d, err := fillReplicationSpecsWithDynamicBlock(resourceb, root)
	if err != nil {
		return err
	}
	if d.IsPresent() {
		resourceb.RemoveBlock(d.block)
		resourceb.SetAttributeRaw(nRepSpecs, d.tokens)
		return nil
	}
	// at least one replication_specs exists here, if not it would be a free tier cluster
	var specbs []*hclwrite.Body
	for {
		var (
			specSrc = resourceb.FirstMatchingBlock(nRepSpecs, nil)
			spec    = hclwrite.NewEmptyFile()
			specb   = spec.Body()
		)
		if specSrc == nil {
			break
		}
		specbSrc := specSrc.Body()
		d, err := fillReplicationSpecsWithDynamicRegionConfigs(specbSrc, root, false)
		if err != nil {
			return err
		}
		if d.IsPresent() {
			resourceb.RemoveBlock(specSrc)
			resourceb.SetAttributeRaw(nRepSpecs, d.tokens)
			return nil
		}
		// ok to fail as zone_name is optional
		_ = hcl.MoveAttr(specbSrc, specb, nZoneName, nZoneName, errRepSpecs)
		shards := specbSrc.GetAttribute(nNumShards)
		if shards == nil {
			return fmt.Errorf("%s: %s not found", errRepSpecs, nNumShards)
		}
		shardsVal, err := hcl.GetAttrInt(shards, errNumShards)
		if err != nil {
			return err
		}
		if err := fillRegionConfigs(specb, specbSrc, root); err != nil {
			return err
		}
		for range shardsVal {
			specbs = append(specbs, specb)
		}
		resourceb.RemoveBlock(specSrc)
	}
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArray(specbs))
	return nil
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
	tokens := hcl.TokensObjectFromExpr(forExpr)
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

func fillBlockOpt(resourceb *hclwrite.Body, name string) {
	block := resourceb.FirstMatchingBlock(name, nil)
	if block == nil {
		return
	}
	resourceb.RemoveBlock(block)
	resourceb.SetAttributeRaw(name, hcl.TokensObject(block.Body()))
}

// fillReplicationSpecsWithDynamicBlock used for dynamic blocks in replication_specs
func fillReplicationSpecsWithDynamicBlock(resourceb *hclwrite.Body, root attrVals) (dynamicBlock, error) {
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs)
	if err != nil || !dSpec.IsPresent() {
		return dynamicBlock{}, err
	}
	transformDynamicBlockReferences(dSpec.content.Body(), nRepSpecs, nSpec)
	dConfig, err := fillReplicationSpecsWithDynamicRegionConfigs(dSpec.content.Body(), root, true)
	if err != nil {
		return dynamicBlock{}, err
	}
	forSpec := hcl.TokensFromExpr(fmt.Sprintf("for %s in %s : ", nSpec, hcl.GetAttrExpr(dSpec.forEach)))
	forSpec = append(forSpec, dConfig.tokens...)
	tokens := hcl.TokensFuncFlatten(forSpec)
	dSpec.tokens = tokens
	return dSpec, nil
}

// fillReplicationSpecsWithDynamicRegionConfigs is used for dynamic blocks in region_configs
func fillReplicationSpecsWithDynamicRegionConfigs(specbSrc *hclwrite.Body, root attrVals, transformRegionReferences bool) (dynamicBlock, error) {
	d, err := getDynamicBlock(specbSrc, nConfigSrc)
	if err != nil || !d.IsPresent() {
		return dynamicBlock{}, err
	}
	repSpec := hclwrite.NewEmptyFile()
	repSpecb := repSpec.Body()
	if zoneName := hcl.GetAttrExpr(specbSrc.GetAttribute(nZoneName)); zoneName != "" {
		repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneName))
	}
	forEach := hcl.GetAttrExpr(d.forEach)
	if transformRegionReferences {
		forEach = replaceDynamicBlockReferences(forEach, nRepSpecs, nSpec)
	}
	regionFor, err := getDynamicBlockRegionConfigsRegionArray(forEach, d.content, root)
	if err != nil {
		return dynamicBlock{}, err
	}
	priorityFor := hcl.TokensComment(commentPriorityFor)
	priorityFor = append(priorityFor, hcl.TokensFromExpr(fmt.Sprintf("for %s in range(%d, %d, -1) : ", nPriority, valMaxPriority, valMinPriority))...)
	priorityFor = append(priorityFor, regionFor...)
	repSpecb.SetAttributeRaw(nConfig, hcl.TokensFuncFlatten(priorityFor))

	shards := specbSrc.GetAttribute(nNumShards)
	if shards == nil {
		return dynamicBlock{}, fmt.Errorf("%s: %s not found", errRepSpecs, nNumShards)
	}
	tokens := hcl.TokensFromExpr(fmt.Sprintf("for i in range(%s) :", hcl.GetAttrExpr(shards)))
	tokens = append(tokens, hcl.EncloseBraces(repSpec.BuildTokens(nil), true)...)
	d.tokens = hcl.EncloseBracketsNewLines(tokens)
	return d, nil
}

func fillRegionConfigs(specb, specbSrc *hclwrite.Body, root attrVals) error {
	var configs []*hclwrite.Body
	for {
		configSrc := specbSrc.FirstMatchingBlock(nConfigSrc, nil)
		if configSrc == nil {
			break
		}
		config, err := getRegionConfig(configSrc, root, false)
		if err != nil {
			return err
		}
		configs = append(configs, config.Body())
		specbSrc.RemoveBlock(configSrc)
	}
	if len(configs) == 0 {
		return fmt.Errorf("%s: %s not found", errRepSpecs, nConfigSrc)
	}
	configs = sortConfigsByPriority(configs)
	specb.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
	return nil
}

func getRegionConfig(configSrc *hclwrite.Block, root attrVals, isDynamicBlock bool) (*hclwrite.File, error) {
	file := hclwrite.NewEmptyFile()
	fileb := file.Body()
	fileb.SetAttributeRaw(nProviderName, root.req[nProviderName])
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nRegionName, nRegionName, errRepSpecs); err != nil {
		return nil, err
	}
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nPriority, nPriority, errRepSpecs); err != nil {
		return nil, err
	}
	if electable, _ := getSpecs(configSrc, nElectableNodes, root, isDynamicBlock); electable != nil {
		fileb.SetAttributeRaw(nElectableSpecs, electable)
	}
	if readOnly, _ := getSpecs(configSrc, nReadOnlyNodes, root, isDynamicBlock); readOnly != nil {
		fileb.SetAttributeRaw(nReadOnlySpecs, readOnly)
	}
	if analytics, _ := getSpecs(configSrc, nAnalyticsNodes, root, isDynamicBlock); analytics != nil {
		fileb.SetAttributeRaw(nAnalyticsSpecs, analytics)
	}
	if autoScaling := getAutoScalingOpt(root.opt); autoScaling != nil {
		fileb.SetAttributeRaw(nAutoScaling, autoScaling)
	}
	return file, nil
}

func getSpecs(configSrc *hclwrite.Block, countName string, root attrVals, isDynamicBlock bool) (hclwrite.Tokens, error) {
	var (
		file  = hclwrite.NewEmptyFile()
		fileb = file.Body()
		count = configSrc.Body().GetAttribute(countName)
	)
	if count == nil {
		return nil, fmt.Errorf("%s: attribute %s not found", errRepSpecs, countName)
	}
	if countVal, errVal := hcl.GetAttrInt(count, errRepSpecs); countVal == 0 && errVal == nil {
		return nil, fmt.Errorf("%s: attribute %s is 0", errRepSpecs, countName)
	}
	fileb.SetAttributeRaw(nNodeCount, count.Expr().BuildTokens(nil))
	fileb.SetAttributeRaw(nInstanceSize, root.req[nInstanceSizeSrc])
	if root.opt[nDiskSizeGB] != nil {
		fileb.SetAttributeRaw(nDiskSizeGB, root.opt[nDiskSizeGB])
	}
	if root.opt[nEBSVolumeTypeSrc] != nil {
		fileb.SetAttributeRaw(nEBSVolumeType, root.opt[nEBSVolumeTypeSrc])
	}
	if root.opt[nDiskIOPSSrc] != nil {
		fileb.SetAttributeRaw(nDiskIOPS, root.opt[nDiskIOPSSrc])
	}
	tokens := hcl.TokensObject(fileb)
	if isDynamicBlock {
		tokens = append(hcl.TokensFromExpr(fmt.Sprintf("%s.%s == 0 ? null :", nRegion, countName)), tokens...)
	}
	return tokens, nil
}

func getAutoScalingOpt(opt map[string]hclwrite.Tokens) hclwrite.Tokens {
	var (
		names = [][2]string{ // use slice instead of map to preserve order
			{nDiskGBEnabledSrc, nDiskGBEnabled},
			{nComputeEnabledSrc, nComputeEnabled},
			{nComputeMinInstanceSizeSrc, nComputeMinInstanceSize},
			{nComputeMaxInstanceSizeSrc, nComputeMaxInstanceSize},
			{nComputeScaleDownEnabledSrc, nComputeScaleDownEnabled},
		}
		file  = hclwrite.NewEmptyFile()
		fileb = file.Body()
		found = false
	)
	for _, tuple := range names {
		src, dst := tuple[0], tuple[1]
		if tokens := opt[src]; tokens != nil {
			fileb.SetAttributeRaw(dst, tokens)
			found = true
		}
	}
	if !found {
		return nil
	}
	return hcl.TokensObject(fileb)
}

func setResourceName(resource *hclwrite.Block, name string) {
	labels := resource.Labels()
	if len(labels) == 0 {
		return
	}
	labels[0] = name
	resource.SetLabels(labels)
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

// getResourceLabel returns the second label of a block, if it exists.
// e.g. in resource "mongodbatlas_cluster" "mycluster", the second label is "mycluster".
func getResourceLabel(resource *hclwrite.Block) string {
	labels := resource.Labels()
	if len(labels) <= 1 {
		return ""
	}
	return labels[1]
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

func checkDynamicBlock(body *hclwrite.Body) error {
	for _, block := range body.Blocks() {
		name := getResourceName(block)
		if block.Type() != nDynamic || slices.Contains(dynamicBlockAllowList, name) {
			continue
		}
		return fmt.Errorf("dynamic blocks are not supported for %s", name)
	}
	return nil
}

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

func replaceDynamicBlockExpr(attr *hclwrite.Attribute, blockName, attrName string) string {
	expr := hcl.GetAttrExpr(attr)
	return strings.ReplaceAll(expr, fmt.Sprintf("%s.%s", blockName, attrName), attrName)
}

// getDynamicBlockRegionConfigsRegionArray returns the region array for a dynamic block in replication_specs.
// e.g. [ for region in var.replication_specs.regions_config : { ... } if priority == region.priority ]
func getDynamicBlockRegionConfigsRegionArray(forEach string, configSrc *hclwrite.Block, root attrVals) (hclwrite.Tokens, error) {
	transformDynamicBlockReferences(configSrc.Body(), nConfigSrc, nRegion)
	priorityStr := hcl.GetAttrExpr(configSrc.Body().GetAttribute(nPriority))
	if priorityStr == "" {
		return nil, fmt.Errorf("%s: %s not found", errRepSpecs, nPriority)
	}
	region, err := getRegionConfig(configSrc, root, true)
	if err != nil {
		return nil, err
	}
	tokens := hcl.TokensFromExpr(fmt.Sprintf("for %s in %s :", nRegion, forEach))
	tokens = append(tokens, hcl.EncloseBraces(region.BuildTokens(nil), true)...)
	tokens = append(tokens, hcl.TokensFromExpr(fmt.Sprintf("if %s == %s", nPriority, priorityStr))...)
	return hcl.EncloseBracketsNewLines(tokens), nil
}

func transformDynamicBlockReferences(configSrcb *hclwrite.Body, blockName, varName string) {
	for name, attr := range configSrcb.Attributes() {
		expr := replaceDynamicBlockReferences(hcl.GetAttrExpr(attr), blockName, varName)
		configSrcb.SetAttributeRaw(name, hcl.TokensFromExpr(expr))
	}
}

// replaceDynamicBlockReferences changes value references, e.g. regions_config.value.electable_nodes to region.electable_nodes
func replaceDynamicBlockReferences(expr, blockName, varName string) string {
	return strings.ReplaceAll(expr,
		fmt.Sprintf("%s.%s.", blockName, nValue),
		fmt.Sprintf("%s.", varName))
}

func sortConfigsByPriority(configs []*hclwrite.Body) []*hclwrite.Body {
	for _, config := range configs {
		if _, err := hcl.GetAttrInt(config.GetAttribute(nPriority), errPriority); err != nil {
			return configs // don't sort priorities if any is not a numerical literal
		}
	}
	sort.Slice(configs, func(i, j int) bool {
		pi, _ := hcl.GetAttrInt(configs[i].GetAttribute(nPriority), errPriority)
		pj, _ := hcl.GetAttrInt(configs[j].GetAttribute(nPriority), errPriority)
		return pi > pj
	})
	return configs
}

func setKeyValue(body *hclwrite.Body, key, value *hclwrite.Attribute) {
	keyStr, err := hcl.GetAttrString(key, "")
	if err == nil {
		if !hclsyntax.ValidIdentifier(keyStr) {
			keyStr = strconv.Quote(keyStr) // wrap in quotes so invalid identifiers (e.g. with blanks) can be used as attribute names
		}
	} else {
		keyStr = strings.TrimSpace(string(key.Expr().BuildTokens(nil).Bytes()))
		keyStr = "(" + keyStr + ")" // wrap in parentheses so non-literal expressions can be used as attribute names
	}
	body.SetAttributeRaw(keyStr, value.Expr().BuildTokens(nil))
}

// popRootAttrs deletes the attributes common to all replication_specs/regions_config and returns them.
func popRootAttrs(body *hclwrite.Body) (attrVals, error) {
	var (
		reqNames = []string{
			nProviderName,
			nInstanceSizeSrc,
		}
		optNames = []string{
			nElectableNodes,
			nReadOnlyNodes,
			nAnalyticsNodes,
			nDiskSizeGB,
			nDiskGBEnabledSrc,
			nComputeEnabledSrc,
			nComputeMinInstanceSizeSrc,
			nComputeMaxInstanceSizeSrc,
			nComputeScaleDownEnabledSrc,
			nEBSVolumeTypeSrc,
			nDiskIOPSSrc,
		}
		req = make(map[string]hclwrite.Tokens)
		opt = make(map[string]hclwrite.Tokens)
	)
	for _, name := range reqNames {
		tokens, err := hcl.PopAttr(body, name, errRepSpecs)
		if err != nil {
			return attrVals{}, err
		}
		req[name] = tokens
	}
	for _, name := range optNames {
		tokens, _ := hcl.PopAttr(body, name, errRepSpecs)
		if tokens != nil {
			opt[name] = tokens
		}
	}
	return attrVals{req: req, opt: opt}, nil
}
