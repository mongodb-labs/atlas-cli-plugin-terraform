resource "resource1" "res1" {
  name = "name1"
}

resource "mongodbatlas_cluster" "free_cluster" {
  count                       = local.use_free_cluster ? 1 : 0
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = "TENANT"
  backing_provider_name       = "AWS"
  provider_region_name        = var.region
  provider_instance_size_name = "M0"
}

data "mongodbatlas_cluster" "cluster2" {
  name = "name4"
}
