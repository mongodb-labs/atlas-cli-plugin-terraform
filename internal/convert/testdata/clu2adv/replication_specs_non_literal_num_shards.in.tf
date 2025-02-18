resource "mongodbatlas_cluster" "multirep" {
  project_id                  = var.project_id
  name                        = "multirep"
  disk_size_gb                = 80
  num_shards                  = 1
  cloud_backup                = false
  cluster_type                = "GEOSHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = var.num_shards # unresolved num_shards
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
}
