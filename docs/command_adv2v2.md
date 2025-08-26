# Convert mongodbatlas_advanced_cluster to Provider 2.0.0 schema

 advancedClusterToV2 (adv2v2) command helps you migrate previous `mongodbatlas_advanced_cluster` configurations to the new Provider 2.0.0 schema.

MongoDB Atlas Provider 2.0.0 introduces a new, cleaner structure for `mongodbatlas_advanced_cluster` resources. The main changes include the use of nested attributes instead of blocks and deletion of deprecated attributes like `disk_size_gb` at root level or `num_shards`.

## Usage

To convert a Terraform configuration from the previous `mongodbatlas_advanced_cluster` schema to the Provider 2.0.0 schema, use the following command:

```bash
atlas terraform advancedClusterToV2 --file in.tf --output out.tf
```

You can also use shorter aliases:
```bash
atlas tf adv2v2 -f in.tf -o out.tf
```

### Command Options

- `--file` or `-f`: Input file path containing the `mongodbatlas_advanced_cluster` configuration
- `--output` or `-o`: Output file path for the converted Provider 2.0.0 configuration
- `--replaceOutput` or `-r`: Overwrite the output file if it exists, or even use the same output file as the input file
- `--watch` or `-w`: Keep the plugin running and watching for changes in the input file

## Examples

You can find [here](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/adv2v2) examples of input files (suffix .in.tf) and the corresponding output files (suffix .out.tf).

## Dynamic Blocks

`dynamic` blocks are used to generate multiple nested blocks based on a set of values. 
Given the different ways of using dynamic blocks, we recommend reviewing the output and making sure it fits your needs.

### Dynamic blocks in tags and labels

You can use `dynamic` blocks for `tags` and `labels`. The plugin assumes that `for_each` has an expression which is evaluated to a `map` of strings.
You can also combine the use of dynamic blocks in `tags` and `labels` with individual blocks in the same cluster definition, e.g.:
```hcl
tags {
	key   = "environment"
	value = var.environment
}
dynamic "tags" {
	for_each = var.tags
	content {
		key   = tags.key
		value = replace(tags.value, "/", "_")
	}
}
```

### Dynamic blocks in regions_config

You can use `dynamic` blocks for `regions_config`. The plugin assumes that `for_each` has an expression which is evaluated to a `list` of objects.

This is an example of how to use dynamic blocks in `region_configs`:
```hcl
replication_specs {
  num_shards = var.replication_specs.num_shards
  zone_name  = var.replication_specs.zone_name # only needed if you're using zones
  dynamic "region_configs" {
    for_each = var.replication_specs.region_configs
    content {
      priority      = region_configs.value.priority
      provider_name = region_configs.value.provider_name
      region_name   = region_configs.value.region_name
      electable_specs {
        instance_size = region_configs.value.instance_size
        node_count    = region_configs.value.electable_node_count
      }
      # read_only_specs, analytics_specs, auto_scaling and analytics_auto_scaling can also be defined
    }
  }
}
```

### Dynamic blocks in replication_specs

You can use `dynamic` blocks for `replication_specs`. The plugin assumes that `for_each` has an expression which is evaluated to a `list` of objects.

This is an example of how to use dynamic blocks in `replication_specs`:
```hcl
dynamic "replication_specs" {
  for_each = var.replication_specs
  content {
    num_shards = replication_specs.value.num_shards
    zone_name  = replication_specs.value.zone_name # only needed if you're using zones
    dynamic "region_configs" {
      for_each = replication_specs.value.region_configs
      priority      = region_configs.value.priority
      provider_name = region_configs.value.provider_name
      region_name   = region_configs.value.region_name
      electable_specs {
        instance_size = region_configs.value.instance_size
        node_count    = region_configs.value.electable_node_count
      }
      # read_only_specs, analytics_specs, auto_scaling and analytics_auto_scaling can also be defined
    }
  }
}
```

### Limitations

If you need to use the plugin for `dynamic` block use cases not yet supported, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues).

#### Dynamic block and individual blocks in the same resource

Dynamic block and individual blocks for `region_configs` or `replication_specs` are not supported at the same time. The recommended way to handle this is to remove the individual `region_configs` or `replication_specs` blocks and use a local `list` variable to add the individual block information to the variable you're using in the `for_each` expression, using [concat](https://developer.hashicorp.com/terraform/language/functions/concat).

Let's see an example with `regions_config`, it is the same idea for `replication_specs`. In the original configuration file, the `mongodb_cluster` resource is used inside a module that receives the `region_configs` elements in a `list` variable and we want to add an additional `region_configs` with a read-only node.
```hcl
variable "replication_specs" {
  type = object({
    num_shards = number
    region_configs  = list(object({
      priority      = number
      provider_name = string
      region_name   = string
      instance_size = string
      electable_node_count = number
      read_only_node_count = number
    }))
  })
}

resource "mongodbatlas_advanced_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "region_configs" {
      for_each = var.replication_specs.region_configs
      priority      = region_configs.value.priority
      provider_name = region_configs.value.provider_name
      region_name   = region_configs.value.region_name      
      electable_specs {
        instance_size = region_configs.value.instance_size
        node_count    = region_configs.value.electable_node_count
      }
      read_only_specs {
        instance_size = region_configs.value.instance_size
        node_count    = region_configs.value.read_only_node_count
      }
    }
    region_configs { # individual region
      instance_size   = "US_EAST_1"
      read_only_nodes = 1
    }
  }
}
```

We modify the configuration file to create an intermediate `local` variable to merge the `region_configs` variable elements and the additional `region_config`:
```hcl
variable "replication_specs" {
  type = object({
    num_shards = number
    region_configs  = list(object({
      priority      = number
      provider_name = string
      region_name   = string
      instance_size = string
      electable_node_count = number
      read_only_node_count = number
    }))
  })
}

locals {
  region_configs_all = concat(
    var.replication_specs.region_configs,
    [
      {
        priority        = 0
        provide_name    = var.provider_aname
        region_name     = "US_EAST_1"
        instance_size   = var.instance_size
        electable_node_count = 0
        read_only_node_count = 1
      },
    ]
  )
}

resource "mongodbatlas_advanced_cluster" "this" {
  project_id                  = var.project_id
  name                        = var.cluster_name
  cluster_type                = var.cluster_type
  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "regions_config" {
      for_each = local.regions_config_all # changed to use the local variable
      priority      = region_configs.value.priority
      provider_name = region_configs.value.provider_name
      region_name   = region_configs.value.region_name      
      electable_specs {
        instance_size = region_configs.value.instance_size
        node_count    = region_configs.value.electable_node_count
      }
      read_only_specs {
        instance_size = region_configs.value.instance_size
        node_count    = region_configs.value.read_only_node_count
      }
    }
  }
}
```
This modified configuration file has the same behavior as the original one, but it doesn't have individual blocks anymore, only the `dynamic` block, so it is supported by the plugin.
