package convert

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/zclconf/go-cty/cty"
)

type attrVals struct {
	req map[string]hclwrite.Tokens
	opt map[string]hclwrite.Tokens
}

// ClusterToAdvancedCluster transforms all mongodbatlas_cluster definitions in a
// Terraform configuration file into mongodbatlas_advanced_cluster schema 2.0.0.
// All other resources and data sources are left untouched.
// Note: hclwrite.Tokens are used instead of cty.Value so expressions with
// interpolations like var.region can be preserved.
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
			addComments(block, false)
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
		err = processFreeTierCluster(blockb)
	} else {
		err = processCluster(blockb)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func isFreeTierCluster(resourceb *hclwrite.Body) bool {
	providerName, _ := hcl.GetAttrString(resourceb.GetAttribute(nProviderName))
	return providerName == nTenant
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

// createDefaultReplicationSpec creates a default replication_specs for clusters without any
// (e.g. upgraded from free tier).
func createDefaultReplicationSpec(resourceb *hclwrite.Body, root attrVals) error {
	resourceb.SetAttributeValue(nClusterType, cty.StringVal(valClusterType))
	configb := hclwrite.NewEmptyFile().Body()
	hcl.SetAttrInt(configb, nPriority, valMaxPriority)
	if err := hcl.MoveAttr(resourceb, configb, nRegionNameSrc, nRegionName, errRoot); err != nil {
		return err
	}
	if providerNameTokens, found := root.req[nProviderName]; found {
		configb.SetAttributeRaw(nProviderName, providerNameTokens)
	}

	electableSpecb := hclwrite.NewEmptyFile().Body()
	if instanceSizeTokens, found := root.req[nInstanceSizeSrc]; found {
		electableSpecb.SetAttributeRaw(nInstanceSize, instanceSizeTokens)
	}
	electableSpecb.SetAttributeValue(nNodeCount, cty.NumberIntVal(valDefaultNodeCount))
	configb.SetAttributeRaw(nElectableSpecs, hcl.TokensObject(electableSpecb))

	repSpecsb := hclwrite.NewEmptyFile().Body()
	repSpecsb.SetAttributeRaw(nConfig, hcl.TokensArraySingle(configb))
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArraySingle(repSpecsb))
	return nil
}

// fillFreeTierCluster is the entry point to convert clusters in free tier
func processFreeTierCluster(resourceb *hclwrite.Body) error {
	resourceb.SetAttributeValue(nClusterType, cty.StringVal(valClusterType))
	configb := hclwrite.NewEmptyFile().Body()
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
	electableSpecb := hclwrite.NewEmptyFile().Body()
	if err := hcl.MoveAttr(resourceb, electableSpecb, nInstanceSizeSrc, nInstanceSize, errFreeCluster); err != nil {
		return err
	}
	configb.SetAttributeRaw(nElectableSpecs, hcl.TokensObject(electableSpecb))
	repSpecsb := hclwrite.NewEmptyFile().Body()
	repSpecsb.SetAttributeRaw(nConfig, hcl.TokensArraySingle(configb))
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArraySingle(repSpecsb))
	return nil
}

// fillCluster is the entry point to convert clusters with replications_specs (all but free tier)
func processCluster(resourceb *hclwrite.Body) error {
	root, errRoot := popRootAttrs(resourceb)
	if errRoot != nil {
		return errRoot
	}
	resourceb.RemoveAttribute(nNumShards) // num_shards in root is not relevant, only in replication_specs
	// ok to fail as cloud_backup is optional
	_ = hcl.MoveAttr(resourceb, resourceb, nCloudBackup, nBackupEnabled, errRepSpecs)
	if err := processRepSpecsCluster(resourceb, root); err != nil {
		return err
	}
	return processCommonOptionalBlocks(resourceb)
}

func processRepSpecsCluster(resourceb *hclwrite.Body, root attrVals) error {
	d, err := processRepSpecsClusterWithDynamicBlock(resourceb, root)
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
		return createDefaultReplicationSpec(resourceb, root)
	}
	dConfig, err := processConfigsWithDynamicRegion(repSpecBlocks[0].Body(), root, false)
	if err != nil {
		return err
	}
	if dConfig.IsPresent() {
		resourceb.SetAttributeRaw(nRepSpecs, dConfig.tokens)
		return nil
	}
	hasVariableShards := hasVariableNumShards(repSpecBlocks)
	var resultTokens []hclwrite.Tokens
	var resultBodies []*hclwrite.Body
	for _, block := range repSpecBlocks {
		specb := hclwrite.NewEmptyFile().Body()
		specbSrc := block.Body()
		_ = hcl.MoveAttr(specbSrc, specb, nZoneName, nZoneName, errRepSpecs)
		shardsAttr := specbSrc.GetAttribute(nNumShards)
		if shardsAttr == nil {
			return fmt.Errorf("%s: %s not found", errRepSpecs, nNumShards)
		}
		if errConfig := processRegionConfigs(specb, specbSrc, root); errConfig != nil {
			return errConfig
		}
		if hasVariableShards {
			resultTokens = append(resultTokens, processNumShardsWhenSomeIsVariable(shardsAttr, specb))
			continue
		}
		shardsVal, err := hcl.GetAttrInt(shardsAttr, errNumShards)
		if err != nil {
			return err
		}
		for range shardsVal {
			resultBodies = append(resultBodies, specb)
		}
	}
	if hasVariableShards {
		resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensFuncConcat(resultTokens...))
	} else {
		resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArray(resultBodies))
	}
	return nil
}

// fillRepSpecsWithDynamicBlock used for dynamic blocks in replication_specs
func processRepSpecsClusterWithDynamicBlock(resourceb *hclwrite.Body, root attrVals) (dynamicBlock, error) {
	dSpec, err := getDynamicBlock(resourceb, nRepSpecs, true)
	if err != nil || !dSpec.IsPresent() {
		return dynamicBlock{}, err
	}
	transformReferences(dSpec.content.Body(), nRepSpecs, nSpec)
	dConfig, err := processConfigsWithDynamicRegion(dSpec.content.Body(), root, true)
	if err != nil {
		return dynamicBlock{}, err
	}
	if dConfig.tokens != nil {
		forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
		forSpec = append(forSpec, dConfig.tokens...)
		tokens := hcl.TokensFuncFlatten(forSpec)
		dSpec.tokens = tokens
		return dSpec, nil
	}

	// Handle static region_configs blocks inside dynamic replication_specs
	specBody := dSpec.content.Body()
	staticConfigs := collectBlocks(specBody, nConfigSrc)
	repSpecb := hclwrite.NewEmptyFile().Body()
	handleZoneName(repSpecb, specBody, nRepSpecs, nSpec)
	var configs []*hclwrite.Body
	for _, configBlock := range staticConfigs {
		config, err := getRegionConfig(configBlock, root, false)
		if err != nil {
			return dynamicBlock{}, err
		}
		configs = append(configs, config)
	}
	configs = sortConfigsByPriority(configs)
	repSpecb.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
	numShardsAttr := specBody.GetAttribute(nNumShards)
	forSpec := hcl.TokensFromExpr(buildForExpr(nSpec, hcl.GetAttrExpr(dSpec.forEach), true))
	numShardsTokens := buildNumShardsTokens(numShardsAttr, repSpecb, nRepSpecs, nSpec)
	dSpec.tokens = hcl.TokensFuncFlatten(append(forSpec, numShardsTokens...))
	return dSpec, nil
}

// fillConfigsWithDynamicRegion is used for dynamic blocks in region_configs
func processConfigsWithDynamicRegion(specbSrc *hclwrite.Body, root attrVals,
	changeReferences bool) (dynamicBlock, error) {
	d, err := getDynamicBlock(specbSrc, nConfigSrc, true)
	if err != nil || !d.IsPresent() {
		return dynamicBlock{}, err
	}
	repSpecb := hclwrite.NewEmptyFile().Body()
	if zoneName := hcl.GetAttrExpr(specbSrc.GetAttribute(nZoneName)); zoneName != "" {
		repSpecb.SetAttributeRaw(nZoneName, hcl.TokensFromExpr(zoneName))
	}
	forEach := hcl.GetAttrExpr(d.forEach)
	if changeReferences {
		forEach = transformReference(forEach, nRepSpecs, nSpec)
	}
	regionFor, err := getDynamicBlockRegionArray(forEach, d.content, root)
	if err != nil {
		return dynamicBlock{}, err
	}
	priorityForStr := buildForExpr(nPriority, fmt.Sprintf("range(%d, %d, -1)", valMaxPriority, valMinPriority), true)
	priorityFor := hcl.TokensComment(commentPriorityFor)
	priorityFor = append(priorityFor, hcl.TokensFromExpr(priorityForStr)...)
	priorityFor = append(priorityFor, regionFor...)
	repSpecb.SetAttributeRaw(nConfig, hcl.TokensFuncFlatten(priorityFor))
	shards := specbSrc.GetAttribute(nNumShards)
	if shards == nil {
		return dynamicBlock{}, fmt.Errorf("%s: %s not found", errRepSpecs, nNumShards)
	}
	tokens := hcl.TokensFromExpr(buildForExpr("i", fmt.Sprintf("range(%s)", hcl.GetAttrExpr(shards)), false))
	tokens = append(tokens, hcl.EncloseBraces(repSpecb.BuildTokens(nil), true)...)
	d.tokens = hcl.EncloseBracketsNewLines(tokens)
	return d, nil
}

func processRegionConfigs(specb, specbSrc *hclwrite.Body, root attrVals) error {
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
		configs = append(configs, config)
		specbSrc.RemoveBlock(configSrc)
	}
	if len(configs) == 0 {
		return fmt.Errorf("%s: %s not found", errRepSpecs, nConfigSrc)
	}
	configs = sortConfigsByPriority(configs)
	specb.SetAttributeRaw(nConfig, hcl.TokensArray(configs))
	return nil
}

func getRegionConfig(configSrc *hclwrite.Block, root attrVals, isDynamicBlock bool) (*hclwrite.Body, error) {
	fileb := hclwrite.NewEmptyFile().Body()
	fileb.SetAttributeRaw(nProviderName, root.req[nProviderName])
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nRegionName, nRegionName, errRepSpecs); err != nil {
		return nil, err
	}
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nPriority, nPriority, errRepSpecs); err != nil {
		return nil, err
	}
	processSpec(fileb, configSrc, nElectableSpecs, nElectableNodes, root, isDynamicBlock)
	processSpec(fileb, configSrc, nReadOnlySpecs, nReadOnlyNodes, root, isDynamicBlock)
	processSpec(fileb, configSrc, nAnalyticsSpecs, nAnalyticsNodes, root, isDynamicBlock)

	if autoScaling := getAutoScalingOpt(root.opt); autoScaling != nil {
		fileb.SetAttributeRaw(nAutoScaling, autoScaling)
	}
	return fileb, nil
}

func processSpec(configb *hclwrite.Body, configSrc *hclwrite.Block,
	specName, countName string, root attrVals, isDynamicBlock bool) {
	var (
		fileb = hclwrite.NewEmptyFile().Body()
		count = configSrc.Body().GetAttribute(countName)
	)
	if count == nil {
		return
	}
	if countVal, errVal := hcl.GetAttrInt(count, errRepSpecs); countVal == 0 && errVal == nil {
		return
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
		tokens = append(hcl.TokensFromExpr(fmt.Sprintf("%s == 0 ? null :", hcl.GetAttrExpr(count))), tokens...)
	}
	configb.SetAttributeRaw(specName, tokens)
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
		fileb = hclwrite.NewEmptyFile().Body()
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

// getResourceLabel returns the second label of a block, if it exists.
// e.g. in resource "mongodbatlas_cluster" "mycluster", the second label is "mycluster".
func getResourceLabel(resource *hclwrite.Block) string {
	labels := resource.Labels()
	if len(labels) <= 1 {
		return ""
	}
	return labels[1]
}

// getDynamicBlockRegionArray returns the region array for a dynamic block in replication_specs.
// e.g. [ for region in var.replication_specs.regions_config : { ... } if priority == region.priority ]
func getDynamicBlockRegionArray(forEach string, configSrc *hclwrite.Block, root attrVals) (hclwrite.Tokens, error) {
	transformReferences(configSrc.Body(), nConfigSrc, nRegion)
	priorityStr := hcl.GetAttrExpr(configSrc.Body().GetAttribute(nPriority))
	if priorityStr == "" {
		return nil, fmt.Errorf("%s: %s not found", errRepSpecs, nPriority)
	}
	region, err := getRegionConfig(configSrc, root, true)
	if err != nil {
		return nil, err
	}
	tokens := hcl.TokensFromExpr(buildForExpr(nRegion, forEach, false))
	tokens = append(tokens, hcl.EncloseBraces(region.BuildTokens(nil), true)...)
	tokens = append(tokens, hcl.TokensFromExpr(fmt.Sprintf("if %s == %s", nPriority, priorityStr))...)
	return hcl.EncloseBracketsNewLines(tokens), nil
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
