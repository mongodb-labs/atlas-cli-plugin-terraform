resource "mongodbatlas_advanced_cluster" "geo" {
  project_id   = var.project_id
  name         = "geo"
  cluster_type = "GEOSHARDED"
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = 2
    region_configs {
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      priority      = 7
      electable_specs {
        node_count    = 3
        instance_size = "M10"
      }
    }
  }
  replication_specs {
    zone_name  = "Zone 2"
    num_shards = 3
    region_configs {
      provider_name = "AWS"
      region_name   = "US_WEST_2"
      priority      = 7
      electable_specs {
        node_count    = 2
        instance_size = "M10"
      }
    }
    region_configs {
      provider_name = "AWS"
      region_name   = "EU_CENTRAL_1"
      priority      = 6
      electable_specs {
        node_count    = 1
        instance_size = "M10"
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "geo" {
  project_id   = var.project_id
  name         = "geo"
  cluster_type = "GEOSHARDED"
  disk_size_gb = 123 # using both num_shards and disk_size_gb
  replication_specs {
    zone_name  = "Zone 1"
    num_shards = 2
    region_configs {
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      priority      = 7
      electable_specs {
        node_count    = 3
        instance_size = "M10"
      }
    }
  }
}
