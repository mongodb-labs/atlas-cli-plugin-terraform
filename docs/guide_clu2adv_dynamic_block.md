# Guide to handle dynamic block limitations in regions_config and replication_specs

The plugin command to convert `mongodbatlas_cluster` resources to `mongodbatlas_advanced_cluster` supports `dynamic` blocks for `regions_config` and `replication_specs`. However, there are some limitations when using `dynamic` blocks in these fields. This guide explains how to handle these limitations.

If you need to use the plugin for use cases not yet supported, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues).

## Dynamic block and individual blocks in the same resource

Dynamic block and individual blocks for `regions_config` or `replication_specs` are not supported at the same time. The recommended way to handle this is to remove the individual `regions_config` or `replication_specs` blocks and use a local variable to add the individual block information to the variable you're using in the `for_each` expression, using [concat](https://developer.hashicorp.com/terraform/language/functions/concat) if you're using a list or [setunion](https://developer.hashicorp.com/terraform/language/functions/setunion) for sets.

Let's see an example with `regions_config`, it is the same for `replication_specs`. In the original configuration file, the `mongodb_cluster` resource is used inside a module that receives the `regions_config` elements in a `list` variable and we want to add an additional `region_config` with a read-only node.
```hcl
variable "replication_specs" {
  type = object({
    num_shards = number
    regions_config = list(object({
      region_name     = string
      electable_nodes = number
      priority        = number
      read_only_nodes = number
    }))
  })
}

resource "mongodbatlas_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  provider_name               = var.provider_name
  provider_instance_size_name = var.provider_instance_size_name
  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "regions_config" {
      for_each = var.replication_specs.regions_config
      content {
        region_name     = regions_config.value.region_name
        electable_nodes = regions_config.value.electable_nodes
        priority        = regions_config.value.priority
        read_only_nodes = regions_config.value.read_only_nodes
      }
    }
    regions_config { # individual region
      region_name     = "US_EAST_1"
      read_only_nodes = 1
    }
  }
}
```

We modify the configuration file to create an intermediate `local` variable to merge the `regions_config` variable elements and the additional `region_config`:
```hcl
variable "replication_specs" {
  type = object({
    num_shards = number
    regions_config = list(object({
      region_name     = string
      electable_nodes = number
      priority        = number
      read_only_nodes = number
    }))
  })
}

locals {
  regions_config_all = concat(
    var.replication_specs.regions_config,
    [
      {
        region_name     = "US_EAST_1"
        electable_nodes = 0
        priority        = 0
        read_only_nodes = 1
      },
    ]
  )
}

resource "mongodbatlas_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  provider_name               = var.provider_name
  provider_instance_size_name = var.provider_instance_size_name
  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "regions_config" {
      for_each = local.regions_config_all # changed to use the local variable
      content {
        region_name     = regions_config.value.region_name
        electable_nodes = regions_config.value.electable_nodes
        priority        = regions_config.value.priority
        read_only_nodes = regions_config.value.read_only_nodes
      }
    }
  }
}
```
This modified configuration file has the same behavior as the original one, but it doesn't have individual blocks anymore, only the `dynamic` block, so it is supported by the plugin.
