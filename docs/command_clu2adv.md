# Convert mongodbatlas_cluster to mongodbatlas_advanced_cluster

This command helps you migrate from `mongodbatlas_cluster` to `mongodbatlas_advanced_cluster` (preview provider 2.0.0).

## Usage

You can find more information in the [Migration Guide: Cluster to Advanced Cluster](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs/guides/cluster-to-advanced-cluster-migration-guide).

**Note**: In order to use the **Preview for MongoDB Atlas Provider 2.0.0** of `mongodbatlas_advanced_cluster`, you need to set the environment variable `MONGODB_ATLAS_PREVIEW_PROVIDER_V2_ADVANCED_CLUSTER` to `true`.

If you want to convert a Terraform configuration from `mongodbatlas_cluster` to `mongodbatlas_advanced_cluster`, use the following command:
```bash
atlas terraform clusterToAdvancedCluster --file in.tf --output out.tf
```

you can also use shorter aliases, e.g.: 
```bash
atlas tf clu2adv -f in.tf -o out.tf
```

### Command Options

- `--file` or `-f`: Input file path containing the `mongodbatlas_cluster` configuration
- `--output` or `-o`: Output file path for the converted `mongodbatlas_advanced_cluster` configuration
- `--includeMoved` or `-m`: Include the `moved blocks` in the output file
- `--replaceOutput` or `-r`: Overwrite the output file if it exists, or even use the same output file as the input file
- `--watch` or `-w`: Keep the plugin running and watching for changes in the input file

## Examples

You can find [here](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/clu2adv) some examples of input files (suffix .in.tf) and the corresponding output files (suffix .out.tf).

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

You can use `dynamic` blocks for `regions_config`. The plugin assumes that `for_each` has an expression which is evaluated to a `list` or `set` of objects. See the [dynamic blocks guide](./guide_clu2adv_dynamic_block.md) to learn more about some limitations.
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

You can use `dynamic` blocks for `replication_specs`. The plugin assumes that `for_each` has an expression which is evaluated to a `list` of objects. See the [dynamic blocks guide](./guide_clu2adv_dynamic_block.md) to learn more about some limitations.
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

## Limitations

- `dynamic` blocks are supported with some [limitations](./guide_clu2adv_dynamic_block.md).