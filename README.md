# Atlas CLI plugin for Terraform's MongoDB Atlas Provider

[![Code Health](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml/badge.svg)](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml)

This repository contains the Atlas CLI plugin for [Terraform's MongoDB Atlas Provider](https://registry.terraform.io/providers/mongodb/mongodbatlas/latest/docs).

## Available Commands

The plugin provides the following commands to help with your Terraform configurations:

### 1. clusterToAdvancedCluster (clu2adv)
Convert `mongodbatlas_cluster` resources to `mongodbatlas_advanced_cluster` format for Provider 2.0.0.

**Quick Start:**
```bash
atlas terraform clusterToAdvancedCluster --file in.tf --output out.tf
# or using alias
atlas tf clu2adv -f in.tf -o out.tf
```

[📖 Full Documentation](./docs/command_clu2adv.md) | [Dynamic Blocks Guide](./docs/guide_clu2adv_dynamic_block.md)

### 2. advancedClusterToV2 (adv2v2)
Convert legacy `mongodbatlas_advanced_cluster` configurations to the new Provider 2.0.0 format with simplified structure.

**Quick Start:**
```bash
atlas terraform advancedClusterToV2 --file in.tf --output out.tf
# or using alias
atlas tf adv2v2 -f in.tf -o out.tf
```

[📖 Full Documentation](./docs/command_adv2v2.md) | [Dynamic Blocks Guide](./docs/guide_adv2v2_dynamic_block.md)

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

## Provider 2.0.0 Preview

**Note**: To use the **Preview for MongoDB Atlas Provider 2.0.0**, set the environment variable:
```bash
export MONGODB_ATLAS_PREVIEW_PROVIDER_V2_ADVANCED_CLUSTER=true
```

## Quick Migration Guide

### From mongodbatlas_cluster to mongodbatlas_advanced_cluster
1. Use `clu2adv` to convert your existing cluster configurations
2. Review the converted output, especially dynamic blocks
3. Test in a development environment
4. Apply to production

[Learn more →](./docs/command_clu2adv.md)

### From Legacy to Provider 2.0.0 Format
1. Use `adv2v2` to update your advanced cluster configurations
2. Verify the flattened `replication_specs` structure
3. Check that `region_configs` is now `config`
4. Test thoroughly before production deployment

[Learn more →](./docs/command_adv2v2.md)

## Examples

Find example conversions in the test data directories:
- [clu2adv examples](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/clu2adv)
- [adv2v2 examples](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/adv2v2)

## Feedback

If you find any issues or have any suggestions, please open an [issue](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues) in this repository.

## Contributing

See our [CONTRIBUTING.md](CONTRIBUTING.md) guide.

## License

MongoDB Atlas CLI is released under the Apache 2.0 license. See [LICENSE.md](LICENSE.md)