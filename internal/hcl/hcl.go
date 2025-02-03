package hcl

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const (
	resourceType                = "resource"
	cluster                     = "mongodbatlas_cluster"
	advCluster                  = "mongodbatlas_advanced_cluster"
	strReplicationSpecs         = "replication_specs"
	strRegionConfigs            = "region_configs"
	strElectableSpecs           = "electable_specs"
	strProviderRegionName       = "provider_region_name"
	strRegionName               = "region_name"
	strProviderName             = "provider_name"
	strBackingProviderName      = "backing_provider_name"
	strProviderInstanceSizeName = "provider_instance_size_name"
	strInstanceSize             = "instance_size"
	strClusterType              = "cluster_type"
	strPriority                 = "priority"

	errFreeCluster = "free cluster (because no " + strReplicationSpecs + ")"
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

		if isFreeTier(resourceBody) {
			if err := fillFreeTier(resourceBody); err != nil {
				return nil, err
			}
		}

		resourceBody.AppendNewline()
		appendComment(resourceBody, "Generated by atlas-cli-plugin-terraform.")
		appendComment(resourceBody, "Please confirm that all references to this resource are updated.")
	}
	return parser.Bytes(), nil
}

func isFreeTier(resourceBody *hclwrite.Body) bool {
	return resourceBody.FirstMatchingBlock(strReplicationSpecs, nil) == nil
}

func fillFreeTier(body *hclwrite.Body) error {
	const (
		valClusterType = "REPLICASET"
		valPriority    = 7
	)
	body.SetAttributeValue(strClusterType, cty.StringVal(valClusterType))
	regionConfig := hclwrite.NewEmptyFile()
	regionConfigBody := regionConfig.Body()
	setAttrInt(regionConfigBody, "priority", valPriority)
	if err := moveAttribute(strProviderRegionName, strRegionName, body, regionConfigBody, errFreeCluster); err != nil {
		return err
	}
	if err := moveAttribute(strProviderName, strProviderName, body, regionConfigBody, errFreeCluster); err != nil {
		return err
	}
	if err := moveAttribute(strBackingProviderName, strBackingProviderName, body, regionConfigBody, errFreeCluster); err != nil {
		return err
	}
	electableSpec := hclwrite.NewEmptyFile()
	if err := moveAttribute(strProviderInstanceSizeName, strInstanceSize, body, electableSpec.Body(), errFreeCluster); err != nil {
		return err
	}
	regionConfigBody.SetAttributeRaw(strElectableSpecs, tokensObject(electableSpec))

	replicationSpec := hclwrite.NewEmptyFile()
	replicationSpec.Body().SetAttributeRaw(strRegionConfigs, tokenArrayObject(regionConfig))
	body.SetAttributeRaw(strReplicationSpecs, tokenArrayObject(replicationSpec))
	return nil
}

func moveAttribute(fromAttrName, toAttrName string, fromBody, toBody *hclwrite.Body, errPrefix string) error {
	attr := fromBody.GetAttribute(fromAttrName)
	if attr == nil {
		return fmt.Errorf("%s: attribute %s not found", errPrefix, fromAttrName)
	}
	fromBody.RemoveAttribute(fromAttrName)
	toBody.SetAttributeRaw(toAttrName, attr.Expr().BuildTokens(nil))
	return nil
}

func setAttrInt(body *hclwrite.Body, attrName string, number int) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenNumberLit, Bytes: []byte(strconv.Itoa(number))},
	}
	body.SetAttributeRaw(attrName, tokens)
}

func tokenArrayObject(file *hclwrite.File) hclwrite.Tokens {
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
