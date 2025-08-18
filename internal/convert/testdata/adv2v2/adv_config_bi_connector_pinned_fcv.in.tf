resource "mongodbatlas_advanced_cluster" "this" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = "REPLICASET"
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "EU_WEST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 3
      }
    }
  }

  advanced_configuration {
    # comments in advanced_configuration are kept
    javascript_enabled = true
  }

  bi_connector_config {
    # comments in bi_connector_config are kept
    enabled         = true
    read_preference = "secondary"
  }

  pinned_fcv {
    # comments in pinned_fcv are kept
    expiration_date = var.fcv_date
  }
}
