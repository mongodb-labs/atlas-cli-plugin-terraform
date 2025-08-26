# Convert mongodbatlas_advanced_cluster to Provider 2.0.0 schema

 advancedClusterToV2 (adv2v2) command helps you migrate previous `mongodbatlas_advanced_cluster` configurations to the new Provider 2.0.0 schema.

## Background

MongoDB Atlas Provider 2.0.0 introduces a new, cleaner structure for `mongodbatlas_advanced_cluster` resources. The main changes include:
- Simplified `replication_specs` structure
- `region_configs` is now `config` (as an array)
- Cleaner handling of `num_shards`
- Better support for dynamic blocks

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

## Dynamic Block Handling

The converter intelligently handles `dynamic` blocks, transforming them to work with the new Provider 2.0.0 structure.

### Dynamic Blocks in replication_specs

When using dynamic blocks for `replication_specs`, the converter transforms them into a flattened structure:

#### Basic Dynamic replication_specs

**Input (Legacy Format):**
```hcl
dynamic "replication_specs" {
  for_each = var.replication_specs
  content {
    num_shards = replication_specs.value.num_shards
    zone_name  = replication_specs.value.zone_name
    region_configs {
      region_name     = replication_specs.value.region_name
      priority        = 7
      provider_name   = "AWS"
      electable_specs {
        instance_size = replication_specs.value.instance_size
        node_count    = 3
      }
    }
  }
}
```

**Output (Provider 2.0.0 Format):**
```hcl
replication_specs = flatten([
  for spec in var.replication_specs : [
    for i in range(spec.num_shards) : {
      zone_name = spec.zone_name
      config = [{
        region_name     = spec.region_name
        priority        = 7
        provider_name   = "AWS"
        electable_specs = {
          instance_size = spec.instance_size
          node_count    = 3
        }
      }]
    }
  ]
])
```

#### Nested Dynamic Blocks

When you have nested dynamic blocks (dynamic `replication_specs` containing dynamic `region_configs`), the converter handles both levels:

**Input:**
```hcl
dynamic "replication_specs" {
  for_each = var.replication_specs
  content {
    num_shards = replication_specs.value.num_shards
    zone_name  = replication_specs.value.zone_name
    dynamic "region_configs" {
      for_each = replication_specs.value.regions
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
        for region in spec.regions : {
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

### Dynamic Blocks in region_configs

Dynamic blocks within `region_configs` (now `config` in Provider 2.0.0) are transformed to use list comprehensions:

**Input:**
```hcl
replication_specs {
  num_shards = 1
  dynamic "region_configs" {
    for_each = var.regions
    content {
      region_name     = region_configs.value.region_name
      priority        = region_configs.value.priority
      provider_name   = "AWS"
      electable_specs {
        instance_size = region_configs.value.instance_size
        node_count    = 3
      }
    }
  }
}
```

**Output:**
```hcl
replication_specs = [{
  config = [
    for region in var.regions : {
      region_name     = region.region_name
      priority        = region.priority
      provider_name   = "AWS"
      electable_specs = {
        instance_size = region.instance_size
        node_count    = 3
      }
    }
  ]
}]
```

### Dynamic Blocks in tags and labels

Dynamic blocks for `tags` and `labels` are converted from block syntax to object syntax:

**Tags Input:**
```hcl
dynamic "tags" {
  for_each = var.tags
  content {
    key   = tags.key
    value = tags.value
  }
}
```

**Tags Output:**
```hcl
tags = {
  for tag in var.tags : tag.key => tag.value
}
```

### Handling num_shards with Dynamic Blocks

The converter properly handles `num_shards` expansion:

1. **For literal num_shards values**: The converter expands the replication spec for each shard
2. **For variable num_shards values**: The converter uses `range()` function to handle the expansion dynamically

**Example with variable num_shards:**
```hcl
# Input
dynamic "replication_specs" {
  for_each = var.specs
  content {
    num_shards = replication_specs.value.shards
    # ... config
  }
}

# Output
replication_specs = flatten([
  for spec in var.specs : [
    for i in range(spec.shards) : {
      # ... config
    }
  ]
])
```

### Mixed Static and Dynamic Blocks

If you have both static and dynamic blocks for `replication_specs` or `region_configs`, the converter handles them separately:

**Input:**
```hcl
replication_specs {
  num_shards = 1
  region_configs {
    region_name = "US_EAST_1"
    # ... config
  }
}

dynamic "replication_specs" {
  for_each = var.additional_specs
  content {
    # ... config
  }
}
```

**Output:**
```hcl
replication_specs = concat(
  [{
    config = [{
      region_name = "US_EAST_1"
      # ... config
    }]
  }],
  flatten([
    for spec in var.additional_specs : [
      # ... transformed dynamic content
    ]
  ])
)
```

## Limitations

### Dynamic Block Limitations

1. **Complex for_each expressions**: While the converter preserves complex expressions in `for_each`, they should be verified after conversion to ensure they work with the new structure.

2. **Custom functions in dynamic blocks**: If you use custom functions or complex conditionals within dynamic blocks, these are preserved but must be tested thoroughly.

3. **Variable references transformation**: The converter updates variable references (e.g., `replication_specs.value` to `spec`), but complex nested references should be reviewed.

4. **Block ordering**: The Provider 2.0.0 schema may handle block ordering differently. Ensure any dependencies on block order are maintained.

### General Limitations

- The converter requires valid HCL syntax in the input file
- Manual review is recommended for complex configurations
- Always run `terraform plan` after conversion to verify the changes

If you encounter use cases not yet supported, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues).

## Best Practices

1. **Review the output**: Always review the converted configuration to ensure it matches your intentions.
2. **Test incrementally**: Test the converted configuration in a development environment before applying to production.
3. **Simplify when possible**: If the converter produces complex nested expressions, consider simplifying them manually for better readability.
4. **Use terraform plan**: Always run `terraform plan` after conversion to verify that the changes are as expected.

## More Examples

You can find more examples of dynamic block conversions in the [test data directory](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/adv2v2), particularly:
- `dynamic_replication_specs.in.tf` / `.out.tf`
- `dynamic_region_configs.in.tf` / `.out.tf`
- `dynamic_tags_labels.in.tf` / `.out.tf`
