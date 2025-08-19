resource "mongodbatlas_advanced_cluster" "no_replication_specs" {
  # missing replication_specs
  project_id   = var.project_id
  name         = "cluster-multi-region"
  cluster_type = "REPLICASET"
}
