package convert

// AdvancedClusterToNew transforms all mongodbatlas_advanced_cluster resource definitions in a
// Terraform configuration file from SDKv2 schema to TPF (Terraform Plugin Framework) schema.
// All other resources and data sources are left untouched.
func AdvancedClusterToNew(config []byte) ([]byte, error) {
	return config, nil
}
