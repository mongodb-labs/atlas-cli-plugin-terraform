resource "resource1" "res1" {
  name = "name1"
}

resource "mongodbatlas_advanced_cluster" "free_cluster" { # comment in the resource
  # comment in own line in the beginning
  count      = local.use_free_cluster ? 1 : 0
  project_id = var.project_id # inline comment kept
  name       = var.cluster_name
  # comment in own line at the end happens before replication_specs
  cluster_type = "REPLICASET"
  replication_specs = [{
    region_configs = [{
      priority              = 7
      region_name           = var.region
      provider_name         = "TENANT"
      backing_provider_name = "AWS"
      electable_specs = {
        instance_size = "M0"
      }
    }]
  }]

  # Generated by atlas-cli-plugin-terraform.
  # Please confirm that all references to this resource are updated.
}

data "mongodbatlas_cluster" "cluster2" {
  name = "name4"
}
