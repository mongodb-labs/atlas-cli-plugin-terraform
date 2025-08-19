resource "mongodbatlas_advanced_cluster" "this" {
  project_id   = "<YOUR-PROJECT-ID>"
  name         = "flex-cluster"
  cluster_type = "REPLICASET"

  replication_specs = [
    {
      region_configs = [
        {
          provider_name         = "FLEX"
          backing_provider_name = "AWS"
          region_name           = "US_EAST_1"
          priority              = 7
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
