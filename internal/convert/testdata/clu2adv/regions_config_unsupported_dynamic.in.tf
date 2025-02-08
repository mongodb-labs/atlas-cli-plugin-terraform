resource "mongodbatlas_cluster" "dynamic_region" {
  project_id                  = var.project_id
  name                        = "dynamic"
  num_shards                  = 1
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"

  replication_specs {
    num_shards = 1
    dynamic "regions_config" {
      for_each = {
        US_WEST_2 = {
          electable_nodes = 3
          priority        = 6
          read_only_nodes = 0
        }
        US_WEST_1 = {
          electable_nodes = 1
          priority        = 5
          read_only_nodes = 0
        }
        US_EAST_1 = {
          electable_nodes = 3
          priority        = 7
          read_only_nodes = 0
        }
      }
      content {
        region_name     = regions_config.key
        electable_nodes = regions_config.value.electable_nodes
        priority        = regions_config.value.priority
        read_only_nodes = regions_config.value.read_only_nodes
      }

    }
  }
}
