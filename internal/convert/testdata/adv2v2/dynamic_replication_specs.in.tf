resource "mongodbatlas_advanced_cluster" "dynamic_replication_specs" {
  lifecycle {
    precondition {
      condition     = !(var.auto_scaling_disk_gb_enabled && var.disk_size > 0)
      error_message = "Must use either auto_scaling_disk_gb_enabled or disk_size, not both."
    }
  }

  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = "GEOSHARDED"

  dynamic "tags" {
    for_each = var.tags
    content {
      key   = tags.key
      value = tags.value
    }
  }
  tags {
    key   = "Tag 2"
    value = "Value 2"
  }

  dynamic "replication_specs" {
    for_each = var.replication_specs
    content {
      num_shards = replication_specs.value.num_shards
      zone_name  = replication_specs.value.zone_name
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
          read_only_specs {
            instance_size = region_configs.value.instance_size
            node_count    = region_configs.value.read_only_node_count
          }
        }
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "all_specs" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = "GEOSHARDED"
  disk_size_gb = 123

  dynamic "replication_specs" {
    for_each = var.replication_specs
    content {
      num_shards = replication_specs.value.num_shards
      zone_name  = replication_specs.value.zone_name
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
          read_only_specs {
            instance_size = region_configs.value.instance_size
            node_count    = region_configs.value.read_only_node_count
          }
          analytics_specs {
            instance_size = region_configs.value.instance_size
            node_count    = region_configs.value.analytics_node_count
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
}

# example of variable for demostration purposes, not used in the conversion
variable "replication_specs" {
  description = "List of replication specifications in mongodbatlas_advanced_cluster format"
  type = list(object({
    num_shards = number
    zone_name  = string
    region_configs = list(object({
      provider_name        = string
      region_name          = string
      instance_size        = string
      electable_node_count = number
      read_only_node_count = number
      analytics_node_count = number
      enable_disk_gb       = bool
      enable_compute       = bool
      priority             = number
    }))
  }))
  default = [
    {
      num_shards = 1
      zone_name  = "Zone A"
      region_configs = [
        {
          provider_name        = "AWS"
          region_name          = "US_EAST_1"
          instance_size        = "M10"
          electable_node_count = 3
          read_only_node_count = 0
          analytics_node_count = 0
          enable_disk_gb       = true
          enable_compute       = false
          priority             = 7
        }
      ]
      }, {
      num_shards = 2
      zone_name  = "Zone B"
      region_configs = [
        {
          provider_name        = "AWS"
          region_name          = "US_WEST_2"
          instance_size        = "M10"
          electable_node_count = 2
          read_only_node_count = 1
          analytics_node_count = 1
          enable_disk_gb       = false
          enable_compute       = true
          priority             = 7
          }, {
          provider_name        = "AWS"
          region_name          = "EU_WEST_1"
          instance_size        = "M10"
          electable_node_count = 1
          read_only_node_count = 0
          analytics_node_count = 0
          enable_disk_gb       = true
          enable_compute       = false
          priority             = 6
        }
      ]
    }
  ]
}
