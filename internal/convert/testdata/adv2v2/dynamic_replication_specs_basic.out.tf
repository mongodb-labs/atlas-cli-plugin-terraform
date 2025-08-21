resource "mongodbatlas_advanced_cluster" "dynamic_replication_specs" {
  project_id        = var.project_id
  name              = var.cluster_name
  cluster_type      = "GEOSHARDED"
  replication_specs = flatten([
    for spec in var.replication_specs : [
      for i in range(spec.num_shards) : {
        zone_name = spec.zone_name
        region_configs = [
          for region in spec.region_configs : {
            provider_name = region.provider_name
            region_name = region.region_name
            priority = region.priority
            electable_specs = {
              instance_size = region.instance_size
              node_count = region.electable_node_count
            }
            read_only_specs = {
              instance_size = region.instance_size
              node_count = region.read_only_node_count
            }
          }
        ]
      }
    ]
  ])

  # Updated by atlas-cli-plugin-terraform, please review the changes.
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
          priority             = 7
          }, {
          provider_name        = "AWS"
          region_name          = "EU_WEST_1"
          instance_size        = "M10"
          electable_node_count = 1
          read_only_node_count = 0
          priority             = 6
        }
      ]
    }
  ]
}
