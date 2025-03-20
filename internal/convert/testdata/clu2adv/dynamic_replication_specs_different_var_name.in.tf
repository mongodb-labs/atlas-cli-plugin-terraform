resource "mongodbatlas_cluster" "different_var_names" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  provider_instance_size_name = var.instance_size
  provider_name               = var.provider_name
  dynamic "replication_specs" {
    for_each = var.my_rep_specs
    content {
      num_shards = replication_specs.value.my_shards
      zone_name  = replication_specs.value.my_zone

      dynamic "regions_config" {
        for_each = replication_specs.value.my_regions
        content {
          electable_nodes = regions_config.value.my_electable_nodes
          priority        = regions_config.value.prio
          region_name     = regions_config.value.my_region_name
        }
      }
    }
  }
}

resource "mongodbatlas_cluster" "different_var_names_no_zone_name" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  provider_instance_size_name = var.instance_size
  provider_name               = var.provider_name
  dynamic "replication_specs" {
    for_each = var.my_rep_specs
    content {
      num_shards = replication_specs.value.my_shards
      dynamic "regions_config" {
        for_each = replication_specs.value.my_regions
        content {
          electable_nodes = regions_config.value.my_electable_nodes
          priority        = regions_config.value.prio
          region_name     = regions_config.value.my_region_name
        }
      }
    }
  }
}
