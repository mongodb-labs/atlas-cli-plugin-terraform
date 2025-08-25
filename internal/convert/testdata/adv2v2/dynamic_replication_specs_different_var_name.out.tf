resource "mongodbatlas_advanced_cluster" "different_var_names" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = var.cluster_type
  replication_specs = flatten([
    for spec in var.my_rep_specs : [
      for i in range(spec.my_shards) : {
        zone_name = spec.my_zone
        region_configs = [
          for region in spec.region_configs : {
            priority      = region.prio
            provider_name = region.provider_name
            region_name   = region.my_region_name
            electable_specs = {
              instance_size = region.instance_size
              node_count    = region.my_electable_node_count
            }
          }
        ]
      }
    ]
  ])

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}

resource "mongodbatlas_advanced_cluster" "different_var_names_no_zone_name_no_num_shards" {
  project_id   = var.project_id
  name         = var.cluster_name
  cluster_type = var.cluster_type
  replication_specs = flatten([
    for spec in var.my_rep_specs : [
      {
        region_configs = [
          for region in spec.region_configs : {
            priority      = region.prio
            provider_name = region.provider_name
            region_name   = region.my_region_name
            electable_specs = {
              instance_size = region.instance_size
              node_count    = region.my_electable_node_count
            }
          }
        ]
      }
    ]
  ])

  # Updated by atlas-cli-plugin-terraform, please review the changes.
}
