resource "mongodbatlas_cluster" "multi_region" {
  project_id   = "1234"
  name         = "cluster-multi-region"
  disk_size_gb = 100
  num_shards   = 1
  cloud_backup = true
  cluster_type = "REPLICASET"

  // Provider Settings "block"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"

  // priorities are not in descending order so regions will be reordered 
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 6
      read_only_nodes = 0
    }
    regions_config {
      region_name     = "US_WEST_1"
      electable_nodes = 1
      priority        = 5
    }
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
      read_only_nodes = 0
    }
  }
}
