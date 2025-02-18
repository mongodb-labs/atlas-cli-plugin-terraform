resource "mongodbatlas_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = var.instance_size
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
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_1"
      electable_nodes = 2
      priority        = 7
    }
  }
}
