# Atlas CLI plugin for Terraform's MongoDB Atlas Provider

[![Code Health](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml/badge.svg)](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml)

This repository contains the Atlas CLI plugin for [Terraform's MongoDB Atlas Provider](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs).

It has the following commands to help with your Terraform configurations:
- **clusterToAdvancedCluster**: Convert a `mongodbatlas_cluster` Terraform configuration to `mongodbatlas_advanced_cluster` (preview provider 2.0.0).

## Installation

Install the [Atlas CLI](https://github.com/mongodb/mongodb-atlas-cli) if you haven't done it yet.

Install the plugin by running:
```bash
atlas plugin install github.com/mongodb-labs/atlas-cli-plugin-terraform
```
 
If you have it installed and want to update it to the latest version, run:
```bash
atlas plugin update mongodb-labs/atlas-cli-plugin-terraform
```

If you want to see the list of installed plugins or check if this plugin is installed, run:
```bash
atlas plugin list
```

## Convert mongodbatlas_cluster to mongodbatlas_advanced_cluster (preview provider 2.0.0)

### Usage

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

If you want to include the `moved blocks` in the output file, use the `--includeMoved` or the `-m` flag.

If you want to overwrite the output file if it exists, or even use the same output file as the input file, use the `--replaceOutput` or the `-r` flag.

You can use the `--watch` or the `-w` flag to keep the plugin running and watching for changes in the input file. You can have input and output files open in an editor and see easily how changes to the input file affect the output file.

You can find [here](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/clu2adv) some examples of input files (suffix .in.tf) and the corresponding output files (suffix .out.tf).

### Dynamic blocks

`dynamic` blocks are used to generate multiple nested blocks based on a set of values. 
Given the different ways of using dynamic blocks, we recommend reviewing the output and making sure it fits your needs.

#### Dynamic blocks in tags and labels

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

#### Dynamic blocks in regions_config

You can use `dynamic` blocks for `regions_config`. The plugin assumes that `for_each` has an expression which is evaluated to a `list` or `set` of objects.
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
Dynamic block and individual blocks for `regions_config` are not supported at the same time. If you need this use case, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues). There are currently two main approaches to handle this:
- (Recommended) Remove the individual `regions_config` blocks and add their information to the variable you're using in the `for_each` expression, e.g. using [concat](https://developer.hashicorp.com/terraform/language/functions/concat) if you're using a list or [setunion](https://developer.hashicorp.com/terraform/language/functions/setunion) for sets. In this way, you don't need to change the generated `mongodb_advanced_cluster` configuration.
- Change the generated `mongodb_advanced_cluster` configuration to join the individual blocks to the code generated for the `dynamic` block. This approach is more error-prone.

#### Dynamic blocks in replication_specs

You can use `dynamic` blocks for `replication_specs`. The plugin assumes that `for_each` has an expression which is evaluated to a `list` of objects.
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
Dynamic block and individual blocks for `replication_specs` are not supported at the same time. If you need this use case, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues). You can handle this following the same approaches as for [`regions_config`](#dynamic-blocks-in-regions_config).

### Limitations

- [`num_shards`](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs/resources/cluster#num_shards-2) in `replication_specs` must be a numeric [literal expression](https://developer.hashicorp.com/nomad/docs/job-specification/hcl2/expressions#literal-expressions), e.g. `var.num_shards` is not supported. This is to allow creating a `replication_specs` element per shard in `mongodbatlas_advanced_cluster`. This limitation doesn't apply if you're using `dynamic` blocks in `regions_config` or `replication_specs`.
- `dynamic` blocks are supported for `tags`, `labels`, `regions_config` and `replication_specs`. See limitations for [`regions_config`](#dynamic-blocks-in-regions_config) and [`replication_specs`](#dynamic-blocks-in-replication_specs) in their sections above.

## Feedback

If you find any issues or have any suggestions, please open an [issue](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues) in this repository.

## Contributing

See our [CONTRIBUTING.md](CONTRIBUTING.md) guide.

## License

MongoDB Atlas CLI is released under the Apache 2.0 license. See [LICENSE.md](LICENSE.md)
