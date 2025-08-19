resource "mongodbatlas_advanced_cluster" "clu" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 2
            disk_size_gb  = 100
          }
        },
        {
          priority      = 6
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          electable_specs = {
            instance_size = "M10"
            node_count    = 1
            disk_size_gb  = 100
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "clu_var" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 2
            disk_size_gb  = var.disk_size_gb
          }
        },
        {
          priority      = 6
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          electable_specs = {
            instance_size = "M10"
            node_count    = 1
            disk_size_gb  = var.disk_size_gb
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "clu_keep" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 2
          }
        },
        {
          priority      = 6
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          electable_specs = {
            disk_size_gb  = 123 # will be kept as root value is not defined
            instance_size = "M10"
            node_count    = 1
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "auto" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 2
            disk_size_gb  = 100
          }
          read_only_specs = {
            instance_size = "M10"
            node_count    = 1
            disk_size_gb  = 100
          }
          analytics_specs = {
            instance_size = "M10"
            node_count    = 1
            disk_size_gb  = 100
          }
          auto_scaling = {
            disk_gb_enabled = true # auto_scaling won't get disk_size_gb
          }
          analytics_auto_scaling = {
            compute_enabled = true # analytics_auto_scaling won't get disk_size_gb
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
