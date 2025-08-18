resource "mongodbatlas_advanced_cluster" "all" {
  project_id   = var.project_id
  name         = "all"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          priority      = 7
          electable_specs = {
            node_count      = 3
            instance_size   = "M10"
            disk_size_gb    = 90
            ebs_volume_type = "PROVISIONED"
            disk_iops       = 100
          }
          read_only_specs = {
            node_count      = 1
            instance_size   = "M10"
            disk_size_gb    = 90
            ebs_volume_type = "PROVISIONED"
            disk_iops       = 100
          }
          analytics_specs = {
            node_count      = 2
            instance_size   = "M10"
            disk_size_gb    = 90
            ebs_volume_type = "PROVISIONED"
            disk_iops       = 100
          }
          auto_scaling = {
            disk_gb_enabled            = true
            compute_enabled            = false
            compute_min_instance_size  = "M10"
            compute_max_instance_size  = "M40"
            compute_scale_down_enabled = local.scale_down
          }
          analytics_auto_scaling = {
            disk_gb_enabled            = false
            compute_enabled            = true
            compute_min_instance_size  = "M20"
            compute_max_instance_size  = "M30"
            compute_scale_down_enabled = local.analytics_scale_down
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "min" {
  project_id   = var.project_id
  name         = "min"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
          }
          read_only_specs = {
            node_count    = 1
            instance_size = "M10"
          }
          analytics_specs = {
            node_count    = 2
            instance_size = "M10"
          }
          auto_scaling = {
            disk_gb_enabled = true
          }
          analytics_auto_scaling = {
            compute_enabled = true
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
