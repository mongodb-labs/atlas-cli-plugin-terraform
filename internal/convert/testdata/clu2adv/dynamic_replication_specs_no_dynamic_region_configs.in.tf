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

resource "mongodbatlas_cluster" "one_config" {
  project_id                   = "123"
  name                         = "cluster"
  provider_name                = "AWS"
  provider_instance_size_name  = "M10"
  disk_size_gb                 = 10
  auto_scaling_disk_gb_enabled = true
  dynamic "replication_specs" {
    for_each = local.replication_specs_list
    content {
      num_shards = 2
      zone_name  = replication_specs.value.zone_name
      regions_config {
        region_name     = replication_specs.value.region_name
        priority        = 7
        electable_nodes = 3
      }
    }
  }
}

resource "mongodbatlas_cluster" "multiple_config" {
  project_id                   = "123"
  name                         = "cluster"
  provider_name                = "AWS"
  provider_instance_size_name  = "M10"
  disk_size_gb                 = 10
  auto_scaling_disk_gb_enabled = true
  dynamic "replication_specs" {
    for_each = local.replication_specs_list
    content {
      num_shards = 2
      zone_name  = replication_specs.value.zone_name
      regions_config {
        region_name     = replication_specs.value.region_name
        priority        = 7
        electable_nodes = 2
      }
      regions_config {
        region_name     = replication_specs.value.region_name
        priority        = 6
        electable_nodes = 1
        read_only_nodes = 4
      }
    }
  }
}
