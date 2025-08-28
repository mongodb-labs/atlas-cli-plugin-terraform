# Convert mongodbatlas_cluster to mongodbatlas_advanced_cluster

The clusterToAdvancedCluster (clu2adv) command helps you migrate from `mongodbatlas_cluster` to `mongodbatlas_advanced_cluster` Provider 2.0.0 schema.

This revised file migrates the Terraform configurations and state to the latest version and doesn't modify the resources deployed in MongoDB Atlas.

You can find more information in the [Migration Guide: Cluster to Advanced Cluster](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs/guides/cluster-to-advanced-cluster-migration-guide).

## Usage

If you want to convert a Terraform configuration from `mongodbatlas_cluster` to `mongodbatlas_advanced_cluster`, use the following command:
```bash
atlas terraform clusterToAdvancedCluster --file in.tf --output out.tf
```

You can also use shorter aliases, for example:
```bash
atlas tf clu2adv -f in.tf -o out.tf
```

### Command Options

- `--file` or `-f`: Input file path containing the `mongodbatlas_cluster` configuration
- `--output` or `-o`: Output file path for the converted `mongodbatlas_advanced_cluster` configuration
- `--replaceOutput` or `-r`: Overwrite the output file if it exists, or even use the same output file as the input file
- `--watch` or `-w`: Keep the plugin running and watching for changes in the input file
- `--includeMoved` or `-m`: Include the `moved blocks` in the output file

## Examples

You can find [here](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/clu2adv) some examples of input files (suffix .in.tf) and the corresponding output files (suffix .out.tf).

## Dynamic Blocks

`dynamic` blocks are used to generate multiple nested blocks based on a set of values. 
We recommend reviewing the output and making sure it fits your needs.

### Dynamic blocks in tags and labels

You can use `dynamic` blocks for `tags` and `labels`. The plugin assumes that the value of `for_each` is an expression which evaluates to a `map` of strings.
You can also combine the use of dynamic blocks in `tags` and `labels` with individual blocks in the same cluster definition, for exaple:
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

You can use `dynamic` blocks for `regions_config`. The plugin assumes that the value of `for_each` is an expression which evaluates to a `list` of objects.

This is an example of how to use dynamic blocks in `regions_config`:
```hcl
replication_specs {
  num_shards = var.replication_specs.num_shards
  zone_name  = var.replication_specs.zone_name # only needed if you're using zones
  dynamic "regions_config" {
    for_each = var.replication_specs.regions_config
    content {
      priority        = regions_config.value.priority
      region_name     = regions_config.value.region_name
      electable_nodes = regions_config.value.electable_nodes
      read_only_nodes = regions_config.value.read_only_nodes
    }
  }
}
```

### Dynamic blocks in replication_specs

You can use `dynamic` blocks for `replication_specs`. The plugin assumes that the value of `for_each` is an expression which evaluates to a `list` of objects.

This is an example of how to use dynamic blocks in `replication_specs`:
```hcl
dynamic "replication_specs" {
  for_each = var.replication_specs
  content {
    num_shards = replication_specs.value.num_shards
    zone_name  = replication_specs.value.zone_name # only needed if you're using zones
    dynamic "regions_config" {
      for_each = replication_specs.value.regions_config
      content {
        electable_nodes = regions_config.value.electable_nodes
        priority        = regions_config.value.priority
        read_only_nodes = regions_config.value.read_only_nodes
        region_name     = regions_config.value.region_name
      }
    }
  }
}
```

### Limitations

If you need to use the plugin for `dynamic` block use cases not yet supported, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues).

#### Combination of blocks with dynamic and inline expressions

Dynamic blocks and individual blocks for `regions_config` or `replication_specs` are not supported at the same time. Remove the individual `regions_config` or `replication_specs` blocks and use a local `list` variable with [concat](https://developer.hashicorp.com/terraform/language/functions/concat) to add the individual block information to the variable you're using in the `for_each` expression.

Let's see an example with `regions_config`, it's the same idea for `replication_specs`. In the original configuration file, the `mongodb_cluster` resource is used inside a module that receives the `regions_config` elements in a `list` variable and we want to add an additional `regions_config` with a read-only node.
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

We modify the configuration file to create an intermediate `local` variable to merge the `regions_config` variable elements and the additional `regions_config`:
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
This modified configuration file has the same behavior as the original one, but it doesn't have individual blocks anymore, only the `dynamic` block, so it's supported by the plugin.
