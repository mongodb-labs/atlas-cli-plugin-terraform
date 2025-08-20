resource "mongodbatlas_advanced_cluster" "clu" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  disk_size_gb = 100
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 2
      }
    }
    region_configs {
      priority      = 6
      provider_name = "AWS"
      region_name   = "US_WEST_2"
      electable_specs {
        instance_size = "M10"
        node_count    = 1
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "clu_var" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  disk_size_gb = var.disk_size_gb
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 2
      }
    }
    region_configs {
      priority      = 6
      provider_name = "AWS"
      region_name   = "US_WEST_2"
      electable_specs {
        disk_size_gb  = 123 # will be ignored and root value will be used instead
        instance_size = "M10"
        node_count    = 1
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "clu_keep" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 2
      }
    }
    region_configs {
      priority      = 6
      provider_name = "AWS"
      region_name   = "US_WEST_2"
      electable_specs {
        disk_size_gb  = 123 # will be kept as root value is not defined
        instance_size = "M10"
        node_count    = 1
      }
    }
  }
}

resource "mongodbatlas_advanced_cluster" "auto" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "SHARDED"
  disk_size_gb = 100
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "US_EAST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 2
      }
      read_only_specs {
        instance_size = "M10"
        node_count    = 1
      }
      analytics_specs {
        instance_size = "M10"
        node_count    = 1
      }
      auto_scaling {
        disk_gb_enabled = true # auto_scaling won't get disk_size_gb
      }
      analytics_auto_scaling {
        compute_enabled = true # analytics_auto_scaling won't get disk_size_gb
      }
    }
  }
}
