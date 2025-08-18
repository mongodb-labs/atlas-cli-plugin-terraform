resource "mongodbatlas_advanced_cluster" "clu" {
  project_id   = var.project_id
  name         = "clu"
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
    environment   = "dev"
    (var.tag_key) = var.tag_value
    "Tag 2"       = "Value 2"
  }
  labels = {
    label1    = "Val label 1"
    "Label 2" = "label val 2"
  }
  timeouts = {
    create = "60m"
  }

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
