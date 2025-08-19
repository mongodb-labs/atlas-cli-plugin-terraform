resource "mongodbatlas_advanced_cluster" "clu" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "REPLICASET"
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "EU_WEST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 3
      }
    }
  }
}
