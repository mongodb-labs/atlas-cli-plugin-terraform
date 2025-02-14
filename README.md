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

## Convert cluster to advanced_cluster v2

### Usage

If you want to convert a Terraform configuration from `mongodbatlas_cluster` to `mongodbatlas_advanced_cluster` schema v2, use the following command:
```bash
atlas terraform clusterToAdvancedCluster --file in.tf --output out.tf
```

you can also use shorter aliases, e.g.: 
```bash
atlas tf clu2adv -f in.tf -o out.tf
```

If you want to overwrite the output file if it exists, or even use the same output file as the input file, use the `--overwriteOutput true` or the `-w` flag.

### Limitations

- The plugin doesn't support `regions_config` without `electable_nodes` as there can be some issues with `priority` when they only have `analytics_nodes` and/or `electable_nodes`.
- `priority` is required in `regions_config` and must be a resolved number between 7 and 1, e.g. `var.priority` is not supported. This is to allow reordering them by descending priority as this is expected in `mongodbatlas_advanced_cluster`.
- `dynamic` blocks to generate `replication_specs`, `regions_config`, etc. are not supported.

## Contributing

See our [CONTRIBUTING.md](CONTRIBUTING.md) guide.

## License

MongoDB Atlas CLI is released under the Apache 2.0 license. See [LICENSE.md](LICENSE.md)
