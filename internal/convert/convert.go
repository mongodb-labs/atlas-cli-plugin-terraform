package convert

import (
	"errors"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/zclconf/go-cty/cty"
)

const (
	resourceType   = "resource"
	cluster        = "mongodbatlas_cluster"
	advCluster     = "mongodbatlas_advanced_cluster"
	valClusterType = "REPLICASET"
	valMaxPriority = 7
	valMinPriority = 1
	errFreeCluster = "free cluster (because no " + nRepSpecs + ")"
	errRepSpecs    = "setting " + nRepSpecs
	errConfigs     = "setting " + nConfig
	errPriority    = "setting " + nPriority
)

type attrVals struct {
	req map[string]hclwrite.Tokens
	opt map[string]hclwrite.Tokens
}

// ClusterToAdvancedCluster transforms all mongodbatlas_cluster definitions in a
// Terraform configuration file into mongodbatlas_advanced_cluster schema v2 definitions.
// All other resources and data sources are left untouched.
// Note: hclwrite.Tokens are used instead of cty.Value so expressions like var.region can be preserved.
// cty.Value only supports resolved values.
func ClusterToAdvancedCluster(config []byte) ([]byte, error) {
	parser, err := hcl.GetParser(config)
	if err != nil {
		return nil, err
	}
	for _, resource := range parser.Body().Blocks() {
		labels := resource.Labels()
		resourceName := labels[0]
		if resource.Type() != resourceType || resourceName != cluster {
			continue
		}
		resourceb := resource.Body()
		if errDyn := checkDynamicBlock(resourceb); errDyn != nil {
			return nil, errDyn
		}
		labels[0] = advCluster
		resource.SetLabels(labels)

		if resourceb.FirstMatchingBlock(nRepSpecs, nil) != nil {
			err = fillReplicationSpecs(resourceb)
		} else {
			err = fillFreeTier(resourceb)
		}
		if err != nil {
			return nil, err
		}

		resourceb.AppendNewline()
		hcl.AppendComment(resourceb, "Generated by atlas-cli-plugin-terraform.")
		hcl.AppendComment(resourceb, "Please confirm that all references to this resource are updated.")
	}
	return parser.Bytes(), nil
}

// fillFreeTier is the entry point to convert clusters in free tier
func fillFreeTier(resourceb *hclwrite.Body) error {
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
	configb.SetAttributeRaw(nElectableSpecs, hcl.TokensObject(electableSpec))

	repSpecs := hclwrite.NewEmptyFile()
	repSpecs.Body().SetAttributeRaw(nConfig, hcl.TokensArraySingle(config))
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArraySingle(repSpecs))
	return nil
}

// fillReplicationSpecs is the entry point to convert clusters with replications_specs (all but free tier)
func fillReplicationSpecs(resourceb *hclwrite.Body) error {
	root, errRoot := popRootAttrs(resourceb)
	if errRoot != nil {
		return errRoot
	}
	resourceb.RemoveAttribute(nNumShards) // num_shards in root is not relevant, only in replication_specs
	// ok to fail as cloud_backup is optional
	_ = hcl.MoveAttr(resourceb, resourceb, nCloudBackup, nBackupEnabled, errRepSpecs)

	// at least one replication_specs exists here, if not it would be a free tier cluster
	repSpecsSrc := resourceb.FirstMatchingBlock(nRepSpecs, nil)
	if err := checkDynamicBlock(repSpecsSrc.Body()); err != nil {
		return err
	}
	configs, errConfigs := getRegionConfigs(repSpecsSrc, root)
	if errConfigs != nil {
		return errConfigs
	}
	repSpecs := hclwrite.NewEmptyFile()
	repSpecs.Body().SetAttributeRaw(nConfig, configs)

	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArraySingle(repSpecs))
	resourceb.RemoveBlock(repSpecsSrc)
	return nil
}

func getRegionConfigs(repSpecsSrc *hclwrite.Block, root attrVals) (hclwrite.Tokens, error) {
	var configs []*hclwrite.File
	for {
		configSrc := repSpecsSrc.Body().FirstMatchingBlock(nConfigSrc, nil)
		if configSrc == nil {
			break
		}
		config, err := getRegionConfig(configSrc, root)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
		repSpecsSrc.Body().RemoveBlock(configSrc)
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("%s: %s not found", errRepSpecs, nConfigSrc)
	}
	sort.Slice(configs, func(i, j int) bool {
		pi, _ := hcl.GetAttrInt(configs[i].Body().GetAttribute(nPriority), errPriority)
		pj, _ := hcl.GetAttrInt(configs[j].Body().GetAttribute(nPriority), errPriority)
		return pi > pj
	})
	return hcl.TokensArray(configs), nil
}

func getRegionConfig(configSrc *hclwrite.Block, root attrVals) (*hclwrite.File, error) {
	file := hclwrite.NewEmptyFile()
	fileb := file.Body()
	fileb.SetAttributeRaw(nProviderName, root.req[nProviderName])
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nRegionName, nRegionName, errRepSpecs); err != nil {
		return nil, err
	}
	if err := setPriority(fileb, configSrc.Body().GetAttribute(nPriority)); err != nil {
		return nil, err
	}
	electableSpecs, errElect := getElectableSpecs(configSrc, root)
	if errElect != nil {
		return nil, errElect
	}
	fileb.SetAttributeRaw(nElectableSpecs, electableSpecs)
	if readOnly := getReadOnlyAnalyticsOpt(nReadOnlyNodes, configSrc, root); readOnly != nil {
		fileb.SetAttributeRaw(nReadOnlySpecs, readOnly)
	}
	if analytics := getReadOnlyAnalyticsOpt(nAnalyticsNodes, configSrc, root); analytics != nil {
		fileb.SetAttributeRaw(nAnalyticsSpecs, analytics)
	}
	if autoScaling := getAutoScalingOpt(root.opt); autoScaling != nil {
		fileb.SetAttributeRaw(nAutoScaling, autoScaling)
	}
	return file, nil
}

func getElectableSpecs(configSrc *hclwrite.Block, root attrVals) (hclwrite.Tokens, error) {
	file := hclwrite.NewEmptyFile()
	fileb := file.Body()
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nElectableNodes, nNodeCount, errRepSpecs); err != nil {
		return nil, err
	}
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
	return hcl.TokensObject(file), nil
}

func getReadOnlyAnalyticsOpt(countName string, configSrc *hclwrite.Block, root attrVals) hclwrite.Tokens {
	var (
		file  = hclwrite.NewEmptyFile()
		fileb = file.Body()
	)
	count := configSrc.Body().GetAttribute(countName)
	if count == nil {
		return nil
	}
	countVal, errVal := hcl.GetAttrInt(count, errRepSpecs)
	// don't include if read_only_nodes or analytics_nodes is 0
	if countVal == 0 && errVal == nil {
		return nil
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
	return hcl.TokensObject(file)
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
		found = false
	)
	for _, tuple := range names {
		src, dst := tuple[0], tuple[1]
		if tokens := opt[src]; tokens != nil {
			file.Body().SetAttributeRaw(dst, tokens)
			found = true
		}
	}
	if !found {
		return nil
	}
	return hcl.TokensObject(file)
}

func checkDynamicBlock(body *hclwrite.Body) error {
	for _, block := range body.Blocks() {
		if block.Type() == "dynamic" {
			return errors.New("dynamic blocks are not supported")
		}
	}
	return nil
}

func setPriority(body *hclwrite.Body, priority *hclwrite.Attribute) error {
	if priority == nil {
		return fmt.Errorf("%s: %s not found", errRepSpecs, nPriority)
	}
	valPrioriy, err := hcl.GetAttrInt(priority, errPriority)
	if err != nil {
		return err
	}
	if valPrioriy < valMinPriority || valPrioriy > valMaxPriority {
		return fmt.Errorf("%s: %s is %d but must be between %d and %d", errPriority, nPriority, valPrioriy, valMinPriority, valMaxPriority)
	}
	hcl.SetAttrInt(body, nPriority, valPrioriy)
	return nil
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
