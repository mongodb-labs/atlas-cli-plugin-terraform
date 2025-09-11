resource "mongodbatlas_cluster" "free_cluster" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = "TENANT"
  backing_provider_name       = "AWS"
  provider_region_name        = var.region
  provider_instance_size_name = "M0"
}

resource "mongodbatlas_cluster" "count" {
  count                       = local.use_free_cluster ? 1 : 0
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = "TENANT"
  backing_provider_name       = "AWS"
  provider_region_name        = var.region
  provider_instance_size_name = "M0"
}


resource "mongodbatlas_cluster" "upgrade_from_free_cluster" {
  # upgraded free cluster to dedicated
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = "AWS" # changed from TENANT to AWS
  provider_instance_size_name = "M10" # changed from M0 to M10
  # removed backing_provider_name = AWS"
  provider_region_name = var.region
}

resource "mongodbatlas_cluster" "upgrade_from_free_cluster_with_variables" {
  # upgraded free cluster to dedicated, using variables in all attributes
  project_id                  = var.project_id
  name                        = var.cluster_name
  provider_name               = var.provider_name
  provider_instance_size_name = var.instance_size
  provider_region_name        = var.region
}

resource "another_resource" "res1" {
  name = "name1"
}
