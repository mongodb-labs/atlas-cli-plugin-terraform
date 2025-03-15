# Atlas CLI plugin for Terraform's MongoDB Atlas Provider

[![Code Health](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml/badge.svg)](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml)

This repository contains the Atlas CLI plugin for [Terraform's MongoDB Atlas Provider](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs).

It has the following commands to help with your Terraform configurations:
- **clusterToAdvancedCluster**: Convert a `mongodbatlas_cluster` Terraform configuration to `mongodbatlas_advanced_cluster` (preview provider v2).

## Installation

Install the [Atlas CLI](https://github.com/mongodb/mongodb-atlas-cli) if you haven't done it yet.

Install the plugin by running:
```bash
atlas plugin install github.com/mongodb-labs/atlas-cli-plugin-terraform
```

## Convert mongodbatlas_cluster to mongodbatlas_advanced_cluster (preview provider 2.0.0)

### Usage

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

### Dynamic blocks

`dynamic` blocks are used to generate multiple nested blocks based on a set of values. 
Given the different ways of using dynamic blocks, we recommend reviewing the output and making sure it fits your needs.

#### Dynamic blocks in tags and labels

You can use `dynamic` blocks for `tags` and `labels`. You can also combine the use of dynamic blocks in `tags` and `labels` with individual blocks in the same cluster definition, e.g.:
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

### Limitations

- The plugin doesn't support `regions_config` without `electable_nodes` as there can be some issues with `priority` when they only have `analytics_nodes` and/or `electable_nodes`.
- [`priority`](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs/resources/cluster#priority-1) is required in `regions_config` and must be a numeric [literal expression](https://developer.hashicorp.com/nomad/docs/job-specification/hcl2/expressions#literal-expressions) between 7 and 1, e.g. `var.priority` is not supported. This is to allow reordering them by descending priority as this is expected in `mongodbatlas_advanced_cluster`.
- [`num_shards`](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs/resources/cluster#num_shards-2) in `replication_specs` must be a numeric [literal expression](https://developer.hashicorp.com/nomad/docs/job-specification/hcl2/expressions#literal-expressions), e.g. `var.num_shards` is not supported. This is to allow creating a `replication_specs` element per shard in `mongodbatlas_advanced_cluster`.
- `dynamic` blocks are currently supported only for `tags` and `labels`. **Coming soon**: support for `replication_specs` and `regions_config`.

## Contributing

See our [CONTRIBUTING.md](CONTRIBUTING.md) guide.

## License

MongoDB Atlas CLI is released under the Apache 2.0 license. See [LICENSE.md](LICENSE.md)
