# Atlas CLI plugin for Terraform's MongoDB Atlas Provider

[![Code Health](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml/badge.svg)](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml)

This repository contains the Atlas CLI plugin for [Terraform's MongoDB Atlas Provider](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs).

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

## Available Commands

The plugin provides the following commands to help with your Terraform configurations:

### 1. clusterToAdvancedCluster (clu2adv)
Convert `mongodbatlas_cluster` resources to `mongodbatlas_advanced_cluster` Provider 2.0.0 schema.

**Quick Start:**
```bash
atlas terraform clusterToAdvancedCluster --file in.tf --output out.tf
# or using alias
atlas tf clu2adv -f in.tf -o out.tf
```

[📖 Full Documentation](./docs/command_clu2adv.md) | [🔄 Migration Guide: Cluster to Advanced Cluster](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs/guides/cluster-to-advanced-cluster-migration-guide)

### 2. advancedClusterToV2 (adv2v2)
Convert previous `mongodbatlas_advanced_cluster` configurations to the new Provider 2.0.0 schema with simplified structure.

**Quick Start:**
```bash
atlas terraform advancedClusterToV2 --file in.tf --output out.tf
# or using alias
atlas tf adv2v2 -f in.tf -o out.tf
```

[📖 Full Documentation](./docs/command_adv2v2.md)

## Feedback

If you find any issues or have any suggestions, please open an [issue](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues) in this repository.

## Contributing

See our [CONTRIBUTING.md](CONTRIBUTING.md) guide.

## License

MongoDB Atlas CLI is released under the Apache 2.0 license. See [LICENSE.md](LICENSE.md)
