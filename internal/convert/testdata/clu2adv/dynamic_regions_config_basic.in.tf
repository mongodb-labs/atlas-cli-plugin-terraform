resource "mongodbatlas_cluster" "dynamic_regions_config" {
  project_id                  = var.project_id
  name                        = "cluster"
  cluster_type                = "SHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = var.replication_specs.num_shards
    zone_name  = var.zone_name
    dynamic "regions_config" {
      for_each = var.replication_specs.regions_config
      content {
        region_name     = regions_config.value.region_name
        electable_nodes = regions_config.value.electable_nodes
        priority        = regions_config.value.prio
        read_only_nodes = regions_config.value.read_only_nodes
      }
    }
  }
}

# example of variable for demostration purposes, not used in the conversion
variable "replication_specs" {
  type = object({
    num_shards = number
    regions_config = set(object({
      region_name     = string
      electable_nodes = number
      prio            = number
      read_only_nodes = number
    }))
  })
  default = {
    num_shards = 3
    regions_config = [
      {
        region_name     = "US_EAST_1"
        electable_nodes = 3
        prio            = 7
        read_only_nodes = 0
      },
      {
        region_name     = "US_WEST_2"
        electable_nodes = 2
        prio            = 6
        read_only_nodes = 1
      }
    ]
  }
}
