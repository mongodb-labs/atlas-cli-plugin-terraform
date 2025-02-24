resource "mongodbatlas_cluster" "cluster1" {
  project_id                  = var.project_id
  name                        = "clu1"
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
}

resource "mongodbatlas_cluster" "cluster2" {
  project_id                  = var.project_id
  name                        = "clu2"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M30"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
    }
  }
}

resource "mongodbatlas_cluster" "count" {
  # count doesn't affect moved blocks, it works in the same way
  count                       = local.create_cluster ? 1 : 0
  project_id                  = var.project_id
  name                        = "count"
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M30"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
    }
  }
}

resource "mongodbatlas_cluster" "forEach" {
  # for_each doesn't affect moved blocks, it works in the same way
  for_each                    = toset(["clu1", "clu2", "clu3"])
  project_id                  = var.project_id
  name                        = each.key
  cluster_type                = "REPLICASET"
  provider_name               = "AWS"
  provider_instance_size_name = "M30"
  replication_specs {
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
    }
  }
}

resource "another_resource" "another" {
  hello = "there"
}
