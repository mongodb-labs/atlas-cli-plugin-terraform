resource "mongodbatlas_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = var.instance_size
  mongo_db_major_version      = var.mongo_db_major_version

  advanced_configuration {
    # comments in advanced_configuration are kept
    javascript_enabled = true
  }
  bi_connector_config {
    # comments in bi_connector_config are kept
    enabled         = true
    read_preference = "secondary"
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
