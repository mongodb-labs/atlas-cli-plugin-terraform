# Atlas CLI plugin for Terraform's MongoDB Atlas Provider

[![Code Health](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml/badge.svg)](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/actions/workflows/code-health.yml)

This repository contains the Atlas CLI plugin for Terraform's MongoDB Atlas Provider.

WIP

## Installing

Install the [Atlas CLI](https://github.com/mongodb/mongodb-atlas-cli) if you haven't done it yet.

Install the plugin by running:
```bash
atlas plugin install github.com/mongodb-labs/atlas-cli-plugin-terraform
```

## Usage

### Convert cluster to advanced_cluster v2
If you want to convert a Terraform configuration from `mongodbatlas_cluster` to `mongodbatlas_advanced_cluster` schema v2, use the following command:
```bash
atlas terraform clusterToAdvancedCluster --file in.tf --output out.tf
```

you can also use shorter alias, e.g.: 
```bash
atlas tf clu2adv -f in.tf -o out.tf
```

If you want to overwrite the output file if it exists, or even use the same output file as the input file, use `--overwriteOutput true` or `-w` flag.


## Contributing

See our [CONTRIBUTING.md](CONTRIBUTING.md) guide.

## License

MongoDB Atlas CLI is released under the Apache 2.0 license. See [LICENSE.md](LICENSE.md)
