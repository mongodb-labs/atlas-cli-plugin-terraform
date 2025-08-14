resource "mongodbatlas_advanced_cluster" "clu" {
  project_id   = "66d979971ec97b7de1ef8777"
  name         = "clu"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            node_count    = 3
            instance_size = "M10"
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
