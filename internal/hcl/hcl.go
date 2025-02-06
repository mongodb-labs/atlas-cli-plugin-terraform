package hcl

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ClusterToAdvancedCluster transforms all mongodbatlas_cluster definitions in a
// Terraform configuration file into mongodbatlas_advanced_cluster schema v2 definitions.
// All other resources and data sources are left untouched.
// Note: hclwrite.Tokens are used instead of cty.Value so expressions like var.region can be preserved.
// cty.Value only supports resolved values.
func ClusterToAdvancedCluster(config []byte) ([]byte, error) {
	parser, err := getParser(config)
	if err != nil {
		return nil, err
	}
	for _, resource := range parser.Body().Blocks() {
		labels := resource.Labels()
		resourceName := labels[0]
		if resource.Type() != resourceType || resourceName != cluster {
			continue
		}
		resourceBody := resource.Body()
		labels[0] = advCluster
		resource.SetLabels(labels)

		if resourceBody.FirstMatchingBlock(nameReplicationSpecs, nil) != nil {
			err = fillReplicationSpecs(resourceBody)
		} else {
			err = fillFreeTier(resourceBody)
		}
		if err != nil {
			return nil, err
		}

		resourceBody.AppendNewline()
		appendComment(resourceBody, "Generated by atlas-cli-plugin-terraform.")
		appendComment(resourceBody, "Please confirm that all references to this resource are updated.")
	}
	return parser.Bytes(), nil
}

func fillFreeTier(body *hclwrite.Body) error {
	body.SetAttributeValue(nameClusterType, cty.StringVal(valClusterType))
	regionConfig := hclwrite.NewEmptyFile()
	regionConfigBody := regionConfig.Body()
	setAttrInt(regionConfigBody, "priority", valPriority)
	if err := moveAttr(body, regionConfigBody, nameProviderRegionName, nameRegionName, errFreeCluster); err != nil {
		return err
	}
	if err := moveAttr(body, regionConfigBody, nameProviderName, nameProviderName, errFreeCluster); err != nil {
		return err
	}
	if err := moveAttr(body, regionConfigBody, nameBackingProviderName, nameBackingProviderName, errFreeCluster); err != nil {
		return err
	}
	electableSpec := hclwrite.NewEmptyFile()
	if err := moveAttr(body, electableSpec.Body(), nameProviderInstanceSizeName, nameInstanceSize, errFreeCluster); err != nil {
		return err
	}
	regionConfigBody.SetAttributeRaw(nameElectableSpecs, tokensObject(electableSpec))

	replicationSpec := hclwrite.NewEmptyFile()
	replicationSpec.Body().SetAttributeRaw(nameRegionConfigs, tokensArrayObject(regionConfig))
	body.SetAttributeRaw(nameReplicationSpecs, tokensArrayObject(replicationSpec))
	return nil
}

func fillReplicationSpecs(body *hclwrite.Body) error {
	root, err := extractRootAttrs(body, errRepSpecs)
	if err != nil {
		return err
	}

	srcReplicationSpecs := body.FirstMatchingBlock(nameReplicationSpecs, nil)
	// srcRegionsConfig := srcReplicationSpecs.Body().FirstMatchingBlock(nameRegionConfigs, nil)
	// regionName := srcRegionsConfig.Body().GetAttribute(nameRegionName)

	body.RemoveAttribute(nameNumShards) // num_shards in root is not relevant, only in replication_specs
	// ok moveAttr to fail as cloud_backup is optional
	_ = moveAttr(body, body, nameCloudBackup, nameBackupEnabled, errRepSpecs)

	electableSpec := hclwrite.NewEmptyFile()
	if root.opt[nameDiskSizeGB] != nil {
		electableSpec.Body().SetAttributeRaw(nameDiskSizeGB, root.opt[nameDiskSizeGB])
	}

	regionsConfig := hclwrite.NewEmptyFile()
	regionsConfigBody := regionsConfig.Body()
	regionsConfigBody.SetAttributeRaw(nameProviderName, root.req[nameProviderName])
	fillAutoScaling(regionsConfigBody, root.opt)
	regionsConfigBody.SetAttributeRaw(nameElectableSpecs, tokensObject(electableSpec))

	replicationSpec := hclwrite.NewEmptyFile()
	replicationSpec.Body().SetAttributeRaw(nameRegionConfigs, tokensArrayObject(regionsConfig))
	body.SetAttributeRaw(nameReplicationSpecs, tokensArrayObject(replicationSpec))

	body.RemoveBlock(srcReplicationSpecs)
	return nil
}

// extractRootAttrs deletes the attributes common to all replication_specs/regions_config and returns them.
func extractRootAttrs(body *hclwrite.Body, errPrefix string) (attrVals, error) {
	var (
		reqNames = []string{
			nameProviderName,
			nameProviderInstanceSizeName,
		}
		optNames = []string{
			nameDiskSizeGB,
			nameAutoScalingDiskGBEnabled,
			nameAutoScalingComputeEnabled,
			nameProviderAutoScalingComputeMinInstanceSize,
			nameProviderAutoScalingComputeMaxInstanceSize,
			nameAutoScalingComputeScaleDownEnabled,
		}
		req = make(map[string]hclwrite.Tokens)
		opt = make(map[string]hclwrite.Tokens)
	)
	for _, name := range reqNames {
		tokens, err := extractAttr(body, name, errPrefix)
		if err != nil {
			return attrVals{}, err
		}
		req[name] = tokens
	}
	for _, name := range optNames {
		tokens, _ := extractAttr(body, name, errPrefix)
		if tokens != nil {
			opt[name] = tokens
		}
	}
	return attrVals{req: req, opt: opt}, nil
}

func fillAutoScaling(regionsConfigBody *hclwrite.Body, opt map[string]hclwrite.Tokens) {
	var (
		names = [][2]string{ // use slice instead of map to preserve order
			{nameAutoScalingDiskGBEnabled, nameDiskGBEnabled},
			{nameAutoScalingComputeEnabled, nameComputeEnabled},
			{nameProviderAutoScalingComputeMinInstanceSize, nameComputeMinInstanceSize},
			{nameProviderAutoScalingComputeMaxInstanceSize, nameComputeMaxInstanceSize},
			{nameAutoScalingComputeScaleDownEnabled, nameComputeScaleDownEnabled},
		}
		file     = hclwrite.NewEmptyFile()
		fileBody = file.Body()
		filled   = false
	)
	for _, tuple := range names {
		oldName, newName := tuple[0], tuple[1]
		if tokens := opt[oldName]; tokens != nil {
			fileBody.SetAttributeRaw(newName, tokens)
			filled = true
		}
	}
	if filled {
		regionsConfigBody.SetAttributeRaw(nameAutoScaling, tokensObject(file))
	}
}

// moveAttr deletes an attribute from fromBody and adds it to toBody.
func moveAttr(fromBody, toBody *hclwrite.Body, fromAttrName, toAttrName, errPrefix string) error {
	tokens, err := extractAttr(fromBody, fromAttrName, errPrefix)
	if err == nil {
		toBody.SetAttributeRaw(toAttrName, tokens)
	}
	return err
}

// extractAttr deletes an attribute and returns it value.
func extractAttr(body *hclwrite.Body, attrName, errPrefix string) (hclwrite.Tokens, error) {
	attr := body.GetAttribute(attrName)
	if attr == nil {
		return nil, fmt.Errorf("%s: attribute %s not found", errPrefix, attrName)
	}
	tokens := attr.Expr().BuildTokens(nil)
	body.RemoveAttribute(attrName)
	return tokens, nil
}

func setAttrInt(body *hclwrite.Body, attrName string, number int) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenNumberLit, Bytes: []byte(strconv.Itoa(number))},
	}
	body.SetAttributeRaw(attrName, tokens)
}

func tokensArrayObject(file *hclwrite.File) hclwrite.Tokens {
	ret := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
	}
	ret = append(ret, tokensObject(file)...)
	ret = append(ret,
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
	return ret
}

func tokensObject(file *hclwrite.File) hclwrite.Tokens {
	ret := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	}
	ret = append(ret, file.BuildTokens(nil)...)
	ret = append(ret,
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("}")})
	return ret
}

func appendComment(body *hclwrite.Body, comment string) {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte("# " + comment + "\n")},
	}
	body.AppendUnstructuredTokens(tokens)
}

func getParser(config []byte) (*hclwrite.File, error) {
	parser, diags := hclwrite.ParseConfig(config, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse Terraform config file: %s", diags.Error())
	}
	return parser, nil
}

type attrVals struct {
	req map[string]hclwrite.Tokens
	opt map[string]hclwrite.Tokens
}

const (
	resourceType = "resource"
	cluster      = "mongodbatlas_cluster"
	advCluster   = "mongodbatlas_advanced_cluster"

	nameReplicationSpecs                          = "replication_specs"
	nameRegionConfigs                             = "region_configs"
	nameElectableSpecs                            = "electable_specs"
	nameAutoScaling                               = "auto_scaling"
	nameProviderRegionName                        = "provider_region_name"
	nameRegionName                                = "region_name"
	nameProviderName                              = "provider_name"
	nameBackingProviderName                       = "backing_provider_name"
	nameProviderInstanceSizeName                  = "provider_instance_size_name"
	nameInstanceSize                              = "instance_size"
	nameClusterType                               = "cluster_type"
	namePriority                                  = "priority"
	nameNumShards                                 = "num_shards"
	nameBackupEnabled                             = "backup_enabled"
	nameCloudBackup                               = "cloud_backup"
	nameDiskSizeGB                                = "disk_size_gb"
	nameAutoScalingDiskGBEnabled                  = "auto_scaling_disk_gb_enabled"
	nameAutoScalingComputeEnabled                 = "auto_scaling_compute_enabled"
	nameAutoScalingComputeScaleDownEnabled        = "auto_scaling_compute_scale_down_enabled"
	nameProviderAutoScalingComputeMinInstanceSize = "provider_auto_scaling_compute_min_instance_size"
	nameProviderAutoScalingComputeMaxInstanceSize = "provider_auto_scaling_compute_max_instance_size"
	nameDiskGBEnabled                             = "disk_gb_enabled"
	nameComputeEnabled                            = "compute_enabled"
	nameComputeScaleDownEnabled                   = "compute_scale_down_enabled"
	nameComputeMinInstanceSize                    = "compute_min_instance_size"
	nameComputeMaxInstanceSize                    = "compute_max_instance_size"

	valClusterType = "REPLICASET"
	valPriority    = 7

	errFreeCluster = "free cluster (because no " + nameReplicationSpecs + ")"
	errRepSpecs    = "setting " + nameReplicationSpecs
)
