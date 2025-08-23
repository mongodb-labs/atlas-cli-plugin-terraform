resource "mongodbatlas_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  mongo_db_major_version      = var.mongo_db_major_version
  provider_instance_size_name = var.instance_size
  provider_name               = var.provider_name

  # dynamic blocks are only supported for tags, labels, replication_specs and regions_config
  dynamic "advanced_configuration" {
    for_each = var.advanced_configuration
    content {
      javascript_enabled = advanced_configuration.value.javascript_enabled
    }
  }
}
