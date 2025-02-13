resource "mongodbatlas_cluster" "basictags" {
  project_id                  = var.project_id
  name                        = "basictags"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  tags {
    key   = "environment"
    value = "dev"
  }
}
