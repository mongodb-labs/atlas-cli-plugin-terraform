resource "mongodbatlas_cluster" "autoscaling" {
  project_id   = var.project_id
  name         = var.cluster_name
  disk_size_gb = 100
  num_shards = 1
  cluster_type = "REPLICASET"

  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
      read_only_nodes = 0
    }
  }
  cloud_backup                            = true
  auto_scaling_disk_gb_enabled            = true
  auto_scaling_compute_enabled            = false
  auto_scaling_compute_scale_down_enabled = local.scale_down

  //Provider Settings "block"
  provider_name                                   = "AWS"
  provider_auto_scaling_compute_min_instance_size = "M10"
  provider_auto_scaling_compute_max_instance_size = "M40"
  provider_instance_size_name                     = "M20"

  lifecycle { // To simulate if there a new instance size name to avoid scale cluster down to original value
    # Note that provider_instance_size_name won't exist in advanced_cluster so it's an error to refer to it,
    # but plugin doesn't help here.
    ignore_changes = [provider_instance_size_name]
  }
}
