resource "mongodbatlas_advanced_cluster" "dynamic_regions_config" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "SHARDED"
  replication_specs {
    num_shards = var.replication_specs.num_shards
    zone_name  = var.zone_name
    dynamic "region_configs" {
      for_each = var.replication_specs.region_configs
      content {
        priority      = region_configs.value.prio
        provider_name = "AWS"
        region_name   = region_configs.value.region_name
        electable_specs {
          instance_size = region_configs.value.instance_size
          node_count    = region_configs.value.node_count
        }
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "using_disk_size_gb" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "SHARDED"
  disk_size_gb = 123
  replication_specs {
    num_shards = var.replication_specs.num_shards
    zone_name  = var.zone_name
    dynamic "region_configs" {
      for_each = var.replication_specs.region_configs
      content {
        priority      = region_configs.value.prio
        provider_name = "AWS"
        region_name   = region_configs.value.region_name
        electable_specs {
          instance_size = region_configs.value.instance_size
          node_count    = region_configs.value.node_count
        }
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "all_specs" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "SHARDED"
  disk_size_gb = 123
  replication_specs {
    num_shards = var.replication_specs.num_shards
    zone_name  = var.zone_name
    dynamic "region_configs" {
      for_each = var.replication_specs.region_configs
      content {
        priority      = region_configs.value.prio
        provider_name = "AWS"
        region_name   = region_configs.value.region_name
        electable_specs {
          instance_size = region_configs.value.instance_size
          node_count    = region_configs.value.node_count
        }
        read_only_specs {
          instance_size = region_configs.value.instance_size
          node_count    = region_configs.value.node_count_read_only
        }
        analytics_specs {
          instance_size = region_configs.value.instance_size
          node_count    = region_configs.value.node_count_analytics
        }
        auto_scaling {
          disk_gb_enabled = region_configs.value.enable_disk_gb
        }
        analytics_auto_scaling {
          compute_enabled = region_configs.value.enable_compute
        }
      }
    }
  }
}

# example of variable for demostration purposes, not used in the conversion
variable "replication_specs" {
  type = object({
    num_shards = number
    region_configs = list(object({
      prio                 = number
      region_name          = string
      instance_size        = string
      node_count           = number
      node_count_read_only = number
      node_count_analytics = number
      enable_disk_gb       = bool
      enable_compute       = bool
    }))
  })
  default = {
    num_shards = 3
    region_configs = [
      {
        prio                 = 7
        region_name          = "US_EAST_1"
        instance_size        = "M10"
        node_count           = 2
        node_count_read_only = 1
        node_count_analytics = 0
        enable_disk_gb       = true
        enable_compute       = false
      },
      {
        prio                 = 6
        region_name          = "US_WEST_2"
        instance_size        = "M10"
        node_count           = 1
        node_count_read_only = 0
        node_count_analytics = 1
        enable_disk_gb       = false
        enable_compute       = true
      }
    ]
  }
}
