resource "mongodbatlas_cluster" "autoscaling" {
  project_id   = var.project_id
  name         = var.cluster_name
  disk_size_gb = 100
  num_shards   = 1
  cluster_type = "REPLICASET"

  replication_specs {
    num_shards = 1
  }

  //Provider Settings "block"
  provider_name                                   = "AWS"
  provider_auto_scaling_compute_min_instance_size = "M10"
  provider_auto_scaling_compute_max_instance_size = "M40"
  provider_instance_size_name                     = "M20"
}
