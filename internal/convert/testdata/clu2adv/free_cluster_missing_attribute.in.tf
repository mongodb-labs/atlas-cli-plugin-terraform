resource "mongodbatlas_cluster" "free_cluster" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = "TENANT"
  provider_region_name        = var.region
  provider_instance_size_name = "M0"
  # missing backing_provider_name
}
