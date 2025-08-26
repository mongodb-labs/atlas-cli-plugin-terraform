resource "mongodbatlas_cluster" "multiple_blocks" {
  project_id                  = var.project_id
  name                        = "cluster"
  cluster_type                = "SHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "regions_config" {
      for_each = var.replication_specs.regions_config
      content {
        region_name     = regions_config.value.region_name
        electable_nodes = regions_config.value.electable_nodes
        priority        = regions_config.value.prio
        read_only_nodes = regions_config.value.read_only_nodes
      }
    }
    regions_config { # inline block is not allowed with dynamic blocks
      region_name     = "US_EAST_1"
      read_only_nodes = 1
    }
  }
}
