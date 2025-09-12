resource "mongodbatlas_cluster" "free_cluster" { # comment in the resource
  # comment in own line in the beginning
  count      = local.use_free_cluster ? 1 : 0
  project_id = var.project_id # inline comment kept
  name       = var.cluster_name
  # comment in own line in the middle is deleted
  provider_name               = "TENANT" # inline comment for attribute moved is not kept
  backing_provider_name       = "AWS"
  provider_region_name        = var.region
  provider_instance_size_name = "M0"
  # comment in own line at the end happens before replication_specs
}

resource "another_resource" "res1" {
  # comment in own line in the middle is not deleted in unprocessed resource
  name = "name1"
}
