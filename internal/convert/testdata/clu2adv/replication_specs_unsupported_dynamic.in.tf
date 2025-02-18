
resource "mongodbatlas_cluster" "geo" {
  project_id                  = var.project_id
  name                        = "geo"
  cluster_type                = "GEOSHARDED"
  num_shards                  = 1
  provider_name               = "AWS"
  provider_instance_size_name = "M30"

  dynamic "replication_specs" {
    for_each = {
      "Zone 1" = {
        region_name = "US_EAST_1"
      },
      "Zone 2" = {
        region_name = "US_WEST_2"
      }
    }
    content {
      zone_name  = replication_specs.key
      num_shards = 2
      regions_config {
        region_name     = replication_specs.value.region_name
        electable_nodes = 3
        priority        = 7
      }
    }
  }
}
