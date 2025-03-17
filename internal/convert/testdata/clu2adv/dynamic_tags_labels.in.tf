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

resource "mongodbatlas_cluster" "simplified_individual" {
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
  tags { // using individual tags apart from simplified version in dynamic tags
    key   = "tag1"
    value = var.tag1val
  }
  dynamic "tags" {
    for_each = var.tags
    content { // simplified version where var can be used directly
      key   = tags.key
      value = tags.value
    }
  }
  tags {
    key   = "tag 2"
    value = var.tag2val
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
  tags { // using individual tags apart from expressions in dynamic tags
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

resource "mongodbatlas_cluster" "full_example" {
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
  labels {
    key   = "label1"
    value = "label1val"
  }
  labels {
    key   = "label2"
    value = data.my_resource.my_data.value
  }
  dynamic "labels" {
    for_each = local.tags
    content {
      key   = labels.key
      value = labels.value
    }
  }
  tags {
    key   = "environment"
    value = "dev"
  }
  tags {
    key   = var.tag_key # non-literal values are supported and enclosed in parentheses
    value = var.tag_value
  }
  dynamic "tags" {
    for_each = var.tags
    content {
      key   = tags.key
      value = replace(tags.value, "/", "_")
    }
  }
  lifecycle {
    precondition {
      condition     = local.use_new_replication_specs || !(var.auto_scaling_disk_gb_enabled && var.disk_size > 0)
      error_message = "Must use either auto_scaling_disk_gb_enabled or disk_size, not both."
    }
  }
}
