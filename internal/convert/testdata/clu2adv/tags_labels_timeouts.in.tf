resource "mongodbatlas_cluster" "basictags" {
  project_id                  = var.project_id
  name                        = "basictags"
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
  tags {
    key   = "environment"
    value = "dev"
  }
}

resource "mongodbatlas_cluster" "basictimeouts" {
  project_id                  = var.project_id
  name                        = "basictimeouts"
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
  timeouts {
    create = "60m"
  }
}

resource "mongodbatlas_cluster" "all" {
  project_id                  = var.project_id
  name                        = "all"
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
  tags {
    key   = "environment"
    value = "dev"
  }
  tags {
    key   = "Tag 2"
    value = "Value 2"
  }
  labels {
    key   = "label1"
    value = "Val label 1"
  }
  labels {
    key   = "Label 2"
    value = "label val 2"
  }
  timeouts {
    # comments in timeouts are also copied
    create = "60m"
    update = "50m"
    delete = "30m"
  }
}
