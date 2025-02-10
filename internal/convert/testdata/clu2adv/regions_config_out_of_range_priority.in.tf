resource "mongodbatlas_cluster" "clu" {
  project_id                  = "1234"
  name                        = "clu"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"

  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      priority        = 0 # range 1-7
      electable_nodes = 2
    }
  }
}
