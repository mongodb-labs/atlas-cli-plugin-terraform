resource "mongodbatlas_cluster" "simplified" {
  project_id                  = var.project_id
  name                        = "cluster"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  dynamic "tags" {
    for_each = var.tags
    content { // simplified version where var can be used directly
      key   = tags.key
      value = tags.value
    }
  }
}

resource "mongodbatlas_cluster" "expression" {
  project_id                  = var.project_id
  name                        = "cluster"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  dynamic "tags" {
    for_each = local.tags
    content { // using expressions
      key   = tags.key
      value = replace(tags.value, "/", "_")
    }
  }
}

resource "mongodbatlas_cluster" "expression_individual" {
  project_id                  = var.project_id
  name                        = "cluster"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  tags { // using individual tags apart from dynamic tags
    key   = "tag1"
    value = var.tag1val
  }
  dynamic "tags" {
    for_each = var.tags
    content { // using expressions
      key   = tags.key
      value = replace(tags.value, "/", "_")
    }
  }
  tags {
    key   = "tag 2"
    value = var.tag2val
  }
}
