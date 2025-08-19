resource "mongodbatlas_advanced_cluster" "multi_region_no_region_configs" {
  # missing region_configs
  project_id   = var.project_id
  name         = "cluster-multi-region"
  cluster_type = "REPLICASET"

  replication_specs {
  }
}
