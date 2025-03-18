resource "mongodbatlas_cluster" "ar" {
  project_id                  = var.project_id
  name                        = "ar"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      priority        = 7
      electable_nodes = 3
      analytics_nodes = 2
      read_only_nodes = 1
    }
  }
}

resource "mongodbatlas_cluster" "ar_not_electable" {
  project_id                  = var.project_id
  name                        = "ar"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      priority        = 7
      analytics_nodes = 2
      read_only_nodes = 1
    }
  }
}
