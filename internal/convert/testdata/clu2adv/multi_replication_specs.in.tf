resource "mongodbatlas_cluster" "basic" {
  project_id                  = var.project_id
  name                        = "multirep"
  disk_size_gb                = 80
  num_shards                  = 1
  cloud_backup                = false
  cluster_type                = "GEOSHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = 1
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  replication_specs {
    zone_name  = "Zone 2"
    num_shards = 1
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
    }
  }
}

resource "mongodbatlas_cluster" "multiple_numerical_num_shards" {
  project_id                  = "1234"
  name                        = "geo"
  disk_size_gb                = 80
  num_shards                  = 1
  cloud_backup                = false
  cluster_type                = "GEOSHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = 2
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
      read_only_nodes = 0
    }
  }
  replication_specs {
    zone_name  = "Zone 2"
    num_shards = 3
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
      read_only_nodes = 0
    }
  }
}

resource "mongodbatlas_cluster" "variable_num_shards" {
  project_id                  = var.project_id
  name                        = "multirep"
  cluster_type                = "GEOSHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = var.num_shards
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
}

resource "mongodbatlas_cluster" "multiple_variable_num_shards" {
  project_id                  = var.project_id
  name                        = "multirep"
  cluster_type                = "GEOSHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = var.num_shards_rep1
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  replication_specs {
    zone_name  = "Zone 2"
    num_shards = var.num_shards_rep2
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
    }
  }
}

resource "mongodbatlas_cluster" "mix_variable_numerical_num_shards" {
  project_id                  = var.project_id
  name                        = "multirep"
  cluster_type                = "GEOSHARDED"
  provider_name               = "AWS"
  provider_instance_size_name = "M10"
  disk_size_gb                = 80
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = 2
    regions_config {
      region_name     = "US_EAST_1"
      electable_nodes = 3
      priority        = 7
    }
  }
  replication_specs {
    zone_name  = "Zone 2"
    num_shards = var.num_shards_rep2
    regions_config {
      region_name     = "US_WEST_2"
      electable_nodes = 3
      priority        = 7
    }
  }
}
