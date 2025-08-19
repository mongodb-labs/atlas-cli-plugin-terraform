resource "mongodbatlas_advanced_cluster" "this" {
  project_id   = var.project_id
  name         = "cluster-tenant"
  cluster_type = "REPLICASET"

  replication_specs = [
    {
      region_configs = [
        {
          provider_name         = "TENANT"
          backing_provider_name = "AWS"
          region_name           = "US_EAST_1"
          priority              = 7
          electable_specs = {
            instance_size = "M0"
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
