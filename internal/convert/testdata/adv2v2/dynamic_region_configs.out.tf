resource "mongodbatlas_advanced_cluster" "dynamic_regions_config" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "SHARDED"
  replication_specs = [
    for i in range(var.replication_specs.num_shards) : {
      zone_name = var.zone_name
      region_configs = [
        for region in var.replication_specs.region_configs : {
          priority      = region.prio
          provider_name = "AWS"
          region_name   = region.region_name
          electable_specs = {
            instance_size = region.instance_size
            node_count    = region.node_count
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "using_disk_size_gb" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "SHARDED"
  replication_specs = [
    for i in range(var.replication_specs.num_shards) : {
      zone_name = var.zone_name
      region_configs = [
        for region in var.replication_specs.region_configs : {
          priority      = region.prio
          provider_name = "AWS"
          region_name   = region.region_name
          electable_specs = {
            instance_size = region.instance_size
            node_count    = region.node_count
            disk_size_gb  = 123
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "all_specs" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "SHARDED"
  replication_specs = [
    for i in range(var.replication_specs.num_shards) : {
      zone_name = var.zone_name
      region_configs = [
        for region in var.replication_specs.region_configs : {
          priority      = region.prio
          provider_name = "AWS"
          region_name   = region.region_name
          electable_specs = {
            instance_size = region.instance_size
            node_count    = region.node_count
            disk_size_gb  = 123
          }
          read_only_specs = {
            instance_size = region.instance_size
            node_count    = region.node_count_read_only
            disk_size_gb  = 123
          }
          analytics_specs = {
            instance_size = region.instance_size
            node_count    = region.node_count_analytics
            disk_size_gb  = 123
          }
          auto_scaling = {
            disk_gb_enabled = region.enable_disk_gb
          }
          analytics_auto_scaling = {
            compute_enabled = region.enable_compute
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
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
