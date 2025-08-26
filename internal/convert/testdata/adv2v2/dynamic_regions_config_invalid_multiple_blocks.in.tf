resource "mongodbatlas_advanced_cluster" "multiple_blocks" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = var.cluster_type
  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "region_configs" {
      for_each = var.replication_specs.region_configs
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
    region_configs { # inline block is not allowed with dynamic blocks
      priority      = 0
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      read_only_specs {
        instance_size = var.instance_size
        node_count    = 1
      }
    }
  }
}
