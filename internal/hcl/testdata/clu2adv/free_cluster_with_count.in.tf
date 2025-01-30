resource "mongodbatlas_cluster" "project_cluster_free" {
  count                       = local.use_free_cluster ? 1 : 0
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = "TENANT"
  backing_provider_name       = "AWS"
  provider_region_name        = var.region
  provider_instance_size_name = "M0"
}
