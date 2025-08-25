resource "mongodbatlas_advanced_cluster" "different_var_names" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = var.cluster_type
  dynamic "replication_specs" {
    for_each = var.my_rep_specs
    content {
      num_shards = replication_specs.value.my_shards
      zone_name  = replication_specs.value.my_zone

      dynamic "region_configs" {
        for_each = replication_specs.value.my_regions
        content {
          priority      = region_configs.value.prio
          provider_name = region_configs.value.provider_name
          region_name   = region_configs.value.my_region_name
          electable_specs {
            instance_size = region_configs.value.instance_size
            node_count    = region_configs.value.my_electable_node_count
          }
        }
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "different_var_names_no_zone_name_no_num_shards" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = var.cluster_type
  dynamic "replication_specs" {
    for_each = var.my_rep_specs
    content {
      dynamic "region_configs" {
        for_each = replication_specs.value.my_regions
        content {
          priority      = region_configs.value.prio
          provider_name = region_configs.value.provider_name
          region_name   = region_configs.value.my_region_name
          electable_specs {
            instance_size = region_configs.value.instance_size
            node_count    = region_configs.value.my_electable_node_count
          }
        }
      }
    }
  }
}
