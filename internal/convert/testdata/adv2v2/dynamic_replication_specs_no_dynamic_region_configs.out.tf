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

  replication_specs = flatten([
    for spec in local.replication_specs_list : [
      for i in range(2) : {
        zone_name = spec.zone_name
        region_configs = [
          {
            priority      = 7
            provider_name = "AWS"
            region_name   = spec.region_name
            electable_specs = {
              instance_size = "M10"
              node_count    = 3
            }
            auto_scaling = {
              disk_gb_enabled = true
            }
          }
        ]
      }
    ]
  ])

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "multiple_config" {
  project_id   = "123"
  name         = "cluster"
  cluster_type = "SHARDED"

  replication_specs = flatten([
    for spec in local.replication_specs_list : [
      for i in range(2) : {
        zone_name = spec.zone_name
        region_configs = [
          {
            priority      = 7
            provider_name = "AWS"
            region_name   = spec.region_name
            electable_specs = {
              instance_size = "M10"
              node_count    = 2
            }
            auto_scaling = {
              disk_gb_enabled = true
            }
          },
          {
            priority      = 6
            provider_name = "AWS"
            region_name   = spec.region_name
            electable_specs = {
              instance_size = "M10"
              node_count    = 1
            }
            auto_scaling = {
              disk_gb_enabled = true
            }
          }
        ]
      }
    ]
  ])

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
