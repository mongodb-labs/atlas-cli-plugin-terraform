# Based on https://github.com/mongodb/terraform-provider-mongodbatlas/blob/master/examples/migrate_cluster_to_advanced_cluster/module_maintainer/v1/main.tf
resource "mongodbatlas_cluster" "this" {
  lifecycle {
    precondition {
      condition     = !(var.auto_scaling_disk_gb_enabled && var.disk_size > 0)
      error_message = "Must use either auto_scaling_disk_gb_enabled or disk_size, not both."
    }
  }

  project_id                   = var.project_id
  name                         = var.cluster_name
  auto_scaling_disk_gb_enabled = var.auto_scaling_disk_gb_enabled
  cluster_type                 = var.cluster_type
  disk_size_gb                 = var.disk_size
  mongo_db_major_version       = var.mongo_db_major_version
  provider_instance_size_name  = var.instance_size
  provider_name                = var.provider_name

  dynamic "tags" {
    for_each = var.tags
    content {
      key   = tags.key
      value = tags.value
    }
  }

  dynamic "replication_specs" {
    for_each = var.replication_specs
    content {
      num_shards = replication_specs.value.num_shards
      zone_name  = replication_specs.value.zone_name

      dynamic "regions_config" {
        for_each = replication_specs.value.regions_config
        content {
          electable_nodes = regions_config.value.electable_nodes
          priority        = regions_config.value.priority
          read_only_nodes = regions_config.value.read_only_nodes
          region_name     = regions_config.value.region_name
        }
      }
    }
  }
}

# example of variable for demostration purposes, not used in the conversion
variable "replication_specs" {
  description = "List of replication specifications in mongodbatlas_cluster format"
  type = list(object({
    num_shards = number
    zone_name  = string
    regions_config = list(object({
      region_name     = string
      electable_nodes = number
      priority        = number
      read_only_nodes = optional(number, 0)
    }))
  }))
  default = [
    {
      num_shards = 1
      zone_name  = "Zone 1"
      regions_config = [
        {
          region_name     = "US_EAST_1"
          electable_nodes = 3
          priority        = 7
        }
      ]
      }, {
      num_shards = 2
      zone_name  = "Zone 2"
      regions_config = [
        {
          region_name     = "US_WEST_2"
          electable_nodes = 2
          priority        = 6
          read_only_nodes = 1
          }, {
          region_name     = "EU_WEST_1"
          electable_nodes = 3
          priority        = 7
        }
      ]
    }
  ]
}
