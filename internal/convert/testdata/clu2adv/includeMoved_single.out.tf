resource "mongodbatlas_advanced_cluster" "cluster" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          provider_name = "AWS"
          region_name   = "US_EAST_1"
          priority      = 7
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
          }
        }
      ]
    }
  ]

  # Generated by atlas-cli-plugin-terraform.
  # Please confirm that all references to this resource are updated.
}

# Moved blocks
# Note: Remember to remove or coment out the old cluster definitions.

moved {
  from = mongodbatlas_cluster.cluster
  to   = mongodbatlas_advanced_cluster.cluster
}
