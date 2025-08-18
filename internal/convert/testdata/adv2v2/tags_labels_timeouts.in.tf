resource "mongodbatlas_advanced_cluster" "clu" {
  project_id   = var.project_id
  name         = "clu"
  cluster_type = "REPLICASET"
  replication_specs {
    region_configs {
      priority      = 7
      provider_name = "AWS"
      region_name   = "EU_WEST_1"
      electable_specs {
        instance_size = "M10"
        node_count    = 3
      }
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
