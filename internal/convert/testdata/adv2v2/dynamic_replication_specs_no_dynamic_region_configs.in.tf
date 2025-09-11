locals {
  replication_specs_list = [
    {
      zone_name   = "zone1"
      region_name = "US_EAST_1"
    },
    {
      zone_name   = "zone2"
      region_name = "US_WEST_2"
    }
  ]
}

resource "mongodbatlas_advanced_cluster" "one_config" {
  project_id   = "123"
  name         = "cluster"
  cluster_type = "SHARDED"

  dynamic "replication_specs" {
    for_each = local.replication_specs_list
    content {
      num_shards = 2
      zone_name  = replication_specs.value.zone_name

      region_configs {
        provider_name = "AWS"
        region_name   = replication_specs.value.region_name
        priority      = 7

        electable_specs {
          instance_size = "M10"
          node_count    = 3
        }
        auto_scaling {
          disk_gb_enabled = true
        }
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "multiple_config" {
  project_id   = "123"
  name         = "cluster"
  cluster_type = "SHARDED"

  dynamic "replication_specs" {
    for_each = local.replication_specs_list
    content {
      num_shards = 2
      zone_name  = replication_specs.value.zone_name

      region_configs {
        provider_name = "AWS"
        region_name   = replication_specs.value.region_name
        priority      = 7

        electable_specs {
          instance_size = "M10"
          node_count    = 2
        }
        auto_scaling {
          disk_gb_enabled = true
        }
      }

      region_configs {
        provider_name = "AWS"
        region_name   = replication_specs.value.region_name
        priority      = 6

        electable_specs {
          instance_size = "M10"
          node_count    = 1
        }
        auto_scaling {
          disk_gb_enabled = true
        }
      }
    }
  }
}
