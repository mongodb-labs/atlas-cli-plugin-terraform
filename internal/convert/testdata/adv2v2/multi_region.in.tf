resource "mongodbatlas_advanced_cluster" "multi_region" {
  project_id     = var.project_id
  name           = "cluster-multi-region"
  cluster_type   = "REPLICASET"
  backup_enabled = true

  replication_specs {
    region_configs {
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      priority      = 7
      electable_specs = {
        node_count    = 3
        instance_size = "M10"
        disk_size_gb  = 100
      }
    }
    region_configs {
      provider_name = "AWS"
      region_name   = "US_WEST_2"
      priority      = 6
      electable_specs = {
        node_count    = 3
        instance_size = "M10"
        disk_size_gb  = 100
      }
    }
    region_configs {
      provider_name = "AWS"
      region_name   = "US_WEST_1"
      priority      = 5
      electable_specs = {
        node_count    = 1
        instance_size = "M10"
        disk_size_gb  = 100
      }
    }
  }
}
