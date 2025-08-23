resource "mongodbatlas_advanced_cluster" "this" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "REPLICASET"

  # dynamic blocks are only supported for tags, labels, replication_specs and region_configs
  dynamic "advanced_configuration" {
    for_each = var.advanced_configuration
    content {
      javascript_enabled = advanced_configuration.value.javascript_enabled
    }
  }

  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "EU_WEST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 3
      }
    }
  }
}
