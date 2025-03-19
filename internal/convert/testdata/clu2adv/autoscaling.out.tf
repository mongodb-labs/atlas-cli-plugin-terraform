resource "mongodbatlas_advanced_cluster" "autoscaling" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = "REPLICASET"



  lifecycle { // To simulate if there a new instance size name to avoid scale cluster down to original value
    # Note that provider_instance_size_name won't exist in advanced_cluster so it's an error to refer to it,
    # but plugin doesn't help here.
    ignore_changes = [provider_instance_size_name]
  }
  backup_enabled = true
  replication_specs = [
    {
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_WEST_2"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M20"
            disk_size_gb  = 100
          }
          auto_scaling = {
            disk_gb_enabled            = true
            compute_enabled            = false
            compute_min_instance_size  = "M10"
            compute_max_instance_size  = "M40"
            compute_scale_down_enabled = local.scale_down
          }
        }
      ]
    }
  ]

  # Generated by atlas-cli-plugin-terraform.
  # Please review the changes and confirm that references to this resource are updated.
}
