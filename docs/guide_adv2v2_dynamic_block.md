# Guide to Dynamic Blocks in advancedClusterToV2 Conversion

The plugin command to convert `mongodbatlas_advanced_cluster` resources from legacy SDKv2 format to Provider 2.0.0 format handles `dynamic` blocks intelligently. This guide explains how dynamic blocks are transformed and any limitations you should be aware of.

If you need to use the plugin for use cases not yet supported, please send us [feedback](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/issues).

## Dynamic Blocks in replication_specs

When using dynamic blocks for `replication_specs`, the converter will transform them into a flattened structure that properly handles the Provider 2.0.0 format.

### Basic Dynamic replication_specs

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

### Nested Dynamic Blocks (replication_specs with dynamic region_configs)

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

## Dynamic Blocks in region_configs

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

## Dynamic Blocks in tags and labels

Dynamic blocks for `tags` and `labels` are converted from block syntax to object syntax:

### Tags

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

### Labels

**Input:**
```hcl
dynamic "labels" {
  for_each = var.labels
  content {
    key   = labels.key
    value = labels.value
  }
}
```

**Output:**
```hcl
labels = {
  for label in var.labels : label.key => label.value
}
```

## Handling num_shards with Dynamic Blocks

The converter properly handles `num_shards` expansion when using dynamic blocks:

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

## Mixed Static and Dynamic Blocks

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

## Limitations and Considerations

1. **Complex for_each expressions**: While the converter preserves complex expressions in `for_each`, you should verify that they still work as expected in the new format.

2. **Custom functions**: If you use custom functions or complex conditionals within dynamic blocks, these are preserved but should be tested after conversion.

3. **Variable references**: The converter updates variable references appropriately (e.g., `replication_specs.value` to `spec`), but complex nested references should be reviewed.

4. **Block ordering**: The Provider 2.0.0 format may handle block ordering differently. Ensure that any dependencies on block order are maintained.

## Best Practices

1. **Review the output**: Always review the converted configuration to ensure it matches your intentions.

2. **Test incrementally**: Test the converted configuration in a development environment before applying to production.

3. **Simplify when possible**: If the converter produces complex nested expressions, consider simplifying them manually for better readability.

4. **Use terraform plan**: Always run `terraform plan` after conversion to verify that the changes are as expected.

## Examples

You can find more examples of dynamic block conversions in the [test data directory](https://github.com/mongodb-labs/atlas-cli-plugin-terraform/tree/main/internal/convert/testdata/adv2v2), particularly:
- `dynamic_replication_specs.in.tf` / `.out.tf`
- `dynamic_region_configs.in.tf` / `.out.tf`
- `dynamic_tags_labels.in.tf` / `.out.tf`