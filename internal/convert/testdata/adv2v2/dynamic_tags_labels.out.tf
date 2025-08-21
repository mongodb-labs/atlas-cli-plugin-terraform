resource "mongodbatlas_advanced_cluster" "simplified" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]
  tags = local.tags

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "expression" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]
  tags = {
    for key, value in local.tags : key => replace(value, "/", "_")
  }

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "simplified_individual" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]
  tags = merge(
    var.tags,
    {
      tag1    = var.tag1val
      "tag 2" = var.tag2val
    }
  )

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "expression_individual" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "REPLICASET"
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]
  tags = merge(
    {
      for key, value in var.tags : key => replace(value, "/", "_")
    },
    {
      tag1    = var.tag1val
      "tag 2" = var.tag2val
    }
  )

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "full_example" {
  project_id   = var.project_id
  name         = "cluster"
  cluster_type = "REPLICASET"
  lifecycle {
    precondition {
      condition     = local.use_new_replication_specs || !(var.auto_scaling_disk_gb_enabled && var.disk_size > 0)
      error_message = "Must use either auto_scaling_disk_gb_enabled or disk_size, not both."
    }
  }
  replication_specs = [
    {
      region_configs = [
        {
          priority      = 7
          provider_name = "AWS"
          region_name   = "EU_WEST_1"
          electable_specs = {
            instance_size = "M10"
            node_count    = 3
          }
        }
      ]
    }
  ]
  tags = merge(
    {
      for key, value in var.tags : key => replace(value, "/", "_")
    },
    {
      environment   = "dev"
      (var.tag_key) = var.tag_value
    }
  )
  labels = merge(
    local.tags,
    {
      label1 = "label1val"
      label2 = data.my_resource.my_data.value
    }
  )

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
