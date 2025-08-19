resource "mongodbatlas_advanced_cluster" "this" {
  # updated, doesn't have nested attributes

  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "this" {
  # not updated, replication_specs is not a block, resource already in TPF format
  project_id   = var.project_id
  name         = "this"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]
}

resource "mongodbatlas_advanced_cluster" "this" {
  # not updated, has an attribute instead of block (timeouts)
  timeouts = {
    create = "60m"
  }
}

datasource "mongodbatlas_advanced_cluster" "this" {
  # not updated, data source
}

resource "another_resource" "this" {
  # not updated, not advanced cluster
}
