package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// ClusterToAdvancedCluster transforms all cluster definition in a
// Terraform config file into an advanced_cluster definition.
// All other resources and data sources are left untouched.
// TODO: at the moment it just changes the resource type.
func ClusterToAdvancedCluster(config []byte) ([]byte, error) {
	parser, err := getParser(config)
	if err != nil {
		return nil, err
	}
	body := parser.Body()
	for _, resource := range body.Blocks() {
		isResource := resource.Type() == "resource"
		labels := resource.Labels()
		resourceName := labels[0]
		if !isResource || resourceName != "mongodbatlas_cluster" {
			continue
		}
		// TODO: Do the full transformation
		labels[0] = "mongodbatlas_advanced_cluster"
		resource.SetLabels(labels)
	}
	return parser.Bytes(), nil
}

func getParser(config []byte) (*hclwrite.File, error) {
	parser, diags := hclwrite.ParseConfig(config, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse Terraform config file: %s", diags.Error())
	}
	return parser, nil
}
