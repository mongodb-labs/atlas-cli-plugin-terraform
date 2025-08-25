# Convert mongodbatlas_advanced_cluster to Provider 2.0.0 Format

This command helps you migrate `mongodbatlas_advanced_cluster` configurations from the legacy SDKv2 format to the new Provider 2.0.0 format.

## Background

MongoDB Atlas Provider 2.0.0 introduces a new, cleaner structure for `mongodbatlas_advanced_cluster` resources. The main changes include:
- Simplified `replication_specs` structure
- `region_configs` is now `config` (as an array)
- Cleaner handling of `num_shards`
- Better support for dynamic blocks

## Usage

To convert a Terraform configuration from the legacy `mongodbatlas_advanced_cluster` format to the Provider 2.0.0 format, use the following command:

```bash
atlas terraform advancedClusterToV2 --file in.tf --output out.tf
```

You can also use shorter aliases:
```bash
atlas tf adv2v2 -f in.tf -o out.tf
```

### Command Options

- `--file` or `-f`: Input file path containing the legacy `mongodbatlas_advanced_cluster` configuration
- `--output` or `-o`: Output file path for the converted Provider 2.0.0 configuration
- `--replaceOutput` or `-r`: Overwrite the output file if it exists
- `--watch` or `-w`: Keep the plugin running and watching for changes in the input file

## Examples

You can find [here](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/adv2v2) examples of input files (suffix .in.tf) and the corresponding output files (suffix .out.tf).

### Basic Conversion Example

**Input (Legacy Format):**
```hcl
resource "mongodbatlas_advanced_cluster" "example" {
  project_id   = var.project_id
  name         = "example-cluster"
  cluster_type = "REPLICASET"

  replication_specs {
    num_shards = 1
    region_configs {
      region_name     = "US_EAST_1"
      priority        = 7
      provider_name   = "AWS"
      electable_specs {
        instance_size = "M10"
        node_count    = 3
      }
    }
  }
}
```

**Output (Provider 2.0.0 Format):**
```hcl
resource "mongodbatlas_advanced_cluster" "example" {
  project_id   = var.project_id
  name         = "example-cluster"
  cluster_type = "REPLICASET"

  replication_specs = [{
    config = [{
      region_name     = "US_EAST_1"
      priority        = 7
      provider_name   = "AWS"
      electable_specs = {
        instance_size = "M10"
        node_count    = 3
      }
    }]
  }]
}
```

## Dynamic Blocks

The converter intelligently handles `dynamic` blocks, transforming them to work with the new Provider 2.0.0 structure.

### Dynamic replication_specs

When you have dynamic `replication_specs` blocks, the converter will:
1. Flatten the structure appropriately
2. Transform `region_configs` to `config`
3. Handle `num_shards` correctly

**Input:**
```hcl
dynamic "replication_specs" {
  for_each = var.replication_specs
  content {
    num_shards = replication_specs.value.num_shards
    zone_name  = replication_specs.value.zone_name
    dynamic "region_configs" {
      for_each = replication_specs.value.region_configs
      content {
        region_name     = region_configs.value.region_name
        priority        = region_configs.value.priority
        provider_name   = region_configs.value.provider_name
        electable_specs {
          instance_size = region_configs.value.instance_size
          node_count    = region_configs.value.node_count
        }
      }
    }
  }
}
```

**Output:**
```hcl
replication_specs = flatten([
  for spec in var.replication_specs : [
    for i in range(spec.num_shards) : {
      zone_name = spec.zone_name
      config = [
        for region in spec.region_configs : {
          region_name     = region.region_name
          priority        = region.priority
          provider_name   = region.provider_name
          electable_specs = {
            instance_size = region.instance_size
            node_count    = region.node_count
          }
        }
      ]
    }
  ]
])
```

### Dynamic tags and labels

Dynamic blocks for `tags` and `labels` are preserved but converted to use the new object syntax:

**Input:**
```hcl
dynamic "tags" {
  for_each = var.tags
  content {
    key   = tags.key
    value = tags.value
  }
}
```

**Output:**
```hcl
tags = {
  for tag in var.tags : tag.key => tag.value
}
```

## Sharded Clusters

For sharded clusters (where `num_shards > 1`), the converter correctly expands the replication specs:

**Input:**
```hcl
replication_specs {
  num_shards = 3
  region_configs {
    region_name     = "US_EAST_1"
    priority        = 7
    provider_name   = "AWS"
    electable_specs {
      instance_size = "M10"
      node_count    = 3
    }
  }
}
```

**Output:**
```hcl
replication_specs = [
  {
    config = [{
      region_name     = "US_EAST_1"
      priority        = 7
      provider_name   = "AWS"
      electable_specs = {
        instance_size = "M10"
        node_count    = 3
      }
    }]
  },
  {
    config = [{
      region_name     = "US_EAST_1"
      priority        = 7
      provider_name   = "AWS"
      electable_specs = {
        instance_size = "M10"
        node_count    = 3
      }
    }]
  },
  {
    config = [{
      region_name     = "US_EAST_1"
      priority        = 7
      provider_name   = "AWS"
      electable_specs = {
        instance_size = "M10"
        node_count    = 3
      }
    }]
  }
]
```

## Limitations

- The converter requires valid HCL syntax in the input file
- Complex expressions in dynamic blocks may require manual review
- Custom functions or complex conditionals in `for_each` expressions are preserved but should be tested
- See the [dynamic blocks guide](./guide_adv2v2_dynamic_block.md) for more details on dynamic block handling