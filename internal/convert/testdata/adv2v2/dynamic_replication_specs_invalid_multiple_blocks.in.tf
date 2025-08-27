resource "mongodbatlas_advanced_cluster" "multiple_blocks" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = var.cluster_type
  dynamic "replication_specs" {
    for_each = var.replication_specs
    content {
      num_shards = replication_specs.value.num_shards
      dynamic "region_configs" {
        for_each = replication_specs.value.region_configs
        content {
          priority      = region_configs.value.priority
          provider_name = region_configs.value.provider_name
          region_name   = region_configs.value.region_name
          electable_specs {
            instance_size = region_configs.value.instance_size
            node_count    = region_configs.value.electable_node_count
          }
        }
      }
    }
  }
  replication_specs { # inline block is not allowed with dynamic blocks
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
