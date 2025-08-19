resource "mongodbatlas_advanced_cluster" "multirep" {
  project_id     = var.project_id
  name           = "multirep"
  cluster_type   = "GEOSHARDED"
  backup_enabled = false
  replication_specs = [
    {
      zone_name = "Zone 1"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    },
    {
      zone_name = "Zone 2"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "geo" {
  project_id     = var.project_id
  name           = "geo"
  cluster_type   = "GEOSHARDED"
  backup_enabled = false
  replication_specs = [
    {
      zone_name = "Zone 1"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    },
    {
      zone_name = "Zone 1"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    },
    {
      zone_name = "Zone 2"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    },
    {
      zone_name = "Zone 2"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    },
    {
      zone_name = "Zone 2"
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
            disk_size_gb  = 80
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
