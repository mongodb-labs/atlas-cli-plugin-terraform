package convert

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/zclconf/go-cty/cty"
)

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
	hcl.SetAttrInt(configb, "priority", valPriority)
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
	repSpecs.Body().SetAttributeRaw(nConfig, hcl.TokensArrayObject(config))
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArrayObject(repSpecs))
	return nil
}

// fillReplicationSpecs is the entry point to convert clusters with replications_specs (all but free tier)
func fillReplicationSpecs(resourceb *hclwrite.Body) error {
	root, errRoot := popRootAttrs(resourceb, errRepSpecs)
	if errRoot != nil {
		return errRoot
	}
	repSpecsSrc := resourceb.FirstMatchingBlock(nRepSpecs, nil)
	configSrc := repSpecsSrc.Body().FirstMatchingBlock(nConfigSrc, nil)
	if configSrc == nil {
		return fmt.Errorf("%s: %s not found", errRepSpecs, nConfigSrc)
	}

	resourceb.RemoveAttribute(nNumShards) // num_shards in root is not relevant, only in replication_specs
	// ok to fail as cloud_backup is optional
	_ = hcl.MoveAttr(resourceb, resourceb, nCloudBackup, nBackupEnabled, errRepSpecs)

	config, errConfig := getRegionConfigs(configSrc, root)
	if errConfig != nil {
		return errConfig
	}
	repSpecs := hclwrite.NewEmptyFile()
	repSpecs.Body().SetAttributeRaw(nConfig, config)
	resourceb.SetAttributeRaw(nRepSpecs, hcl.TokensArrayObject(repSpecs))

	resourceb.RemoveBlock(repSpecsSrc)
	return nil
}

func getRegionConfigs(configSrc *hclwrite.Block, root attrVals) (hclwrite.Tokens, error) {
	file := hclwrite.NewEmptyFile()
	fileb := file.Body()
	fileb.SetAttributeRaw(nProviderName, root.req[nProviderName])
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nRegionName, nRegionName, errRepSpecs); err != nil {
		return nil, err
	}
	if err := hcl.MoveAttr(configSrc.Body(), fileb, nPriority, nPriority, errRepSpecs); err != nil {
		return nil, err
	}
	autoScaling := getAutoScalingOpt(root.opt)
	if autoScaling != nil {
		fileb.SetAttributeRaw(nAutoScaling, autoScaling)
	}
	electableSpecs, errElect := getElectableSpecs(configSrc, root)
	if errElect != nil {
		return nil, errElect
	}
	fileb.SetAttributeRaw(nElectableSpecs, electableSpecs)
	return hcl.TokensArrayObject(file), nil
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
	return hcl.TokensObject(file), nil
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

// popRootAttrs deletes the attributes common to all replication_specs/regions_config and returns them.
func popRootAttrs(body *hclwrite.Body, errPrefix string) (attrVals, error) {
	var (
		reqNames = []string{
			nProviderName,
			nInstanceSizeSrc,
		}
		optNames = []string{
			nDiskSizeGB,
			nDiskGBEnabledSrc,
			nComputeEnabledSrc,
			nComputeMinInstanceSizeSrc,
			nComputeMaxInstanceSizeSrc,
			nComputeScaleDownEnabledSrc,
		}
		req = make(map[string]hclwrite.Tokens)
		opt = make(map[string]hclwrite.Tokens)
	)
	for _, name := range reqNames {
		tokens, err := hcl.PopAttr(body, name, errPrefix)
		if err != nil {
			return attrVals{}, err
		}
		req[name] = tokens
	}
	for _, name := range optNames {
		tokens, _ := hcl.PopAttr(body, name, errPrefix)
		if tokens != nil {
			opt[name] = tokens
		}
	}
	return attrVals{req: req, opt: opt}, nil
}

type attrVals struct {
	req map[string]hclwrite.Tokens
	opt map[string]hclwrite.Tokens
}

const (
	resourceType = "resource"
	cluster      = "mongodbatlas_cluster"
	advCluster   = "mongodbatlas_advanced_cluster"

	nRepSpecs                   = "replication_specs"
	nConfig                     = "region_configs"
	nConfigSrc                  = "regions_config"
	nElectableSpecs             = "electable_specs"
	nAutoScaling                = "auto_scaling"
	nRegionNameSrc              = "provider_region_name"
	nRegionName                 = "region_name"
	nProviderName               = "provider_name"
	nBackingProviderName        = "backing_provider_name"
	nInstanceSizeSrc            = "provider_instance_size_name"
	nInstanceSize               = "instance_size"
	nClusterType                = "cluster_type"
	nPriority                   = "priority"
	nNumShards                  = "num_shards"
	nBackupEnabled              = "backup_enabled"
	nCloudBackup                = "cloud_backup"
	nDiskSizeGB                 = "disk_size_gb"
	nDiskGBEnabledSrc           = "auto_scaling_disk_gb_enabled"
	nComputeEnabledSrc          = "auto_scaling_compute_enabled"
	nComputeScaleDownEnabledSrc = "auto_scaling_compute_scale_down_enabled"
	nComputeMinInstanceSizeSrc  = "provider_auto_scaling_compute_min_instance_size"
	nComputeMaxInstanceSizeSrc  = "provider_auto_scaling_compute_max_instance_size"
	nDiskGBEnabled              = "disk_gb_enabled"
	nComputeEnabled             = "compute_enabled"
	nComputeScaleDownEnabled    = "compute_scale_down_enabled"
	nComputeMinInstanceSize     = "compute_min_instance_size"
	nComputeMaxInstanceSize     = "compute_max_instance_size"
	nNodeCount                  = "node_count"
	nElectableNodes             = "electable_nodes"

	valClusterType = "REPLICASET"
	valPriority    = 7

	errFreeCluster = "free cluster (because no " + nRepSpecs + ")"
	errRepSpecs    = "setting " + nRepSpecs
)
