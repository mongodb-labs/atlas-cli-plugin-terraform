resource "mongodbatlas_cluster" "cluster" {
  project_id                     = var.project_id
  name                           = var.cluster_name
  cloud_backup                   = var.backup_enabled
  pit_enabled                    = var.pit_enabled
  retain_backups_enabled         = var.retain_backups_enabled
  auto_scaling_disk_gb_enabled   = var.auto_scaling_disk_gb_enabled
  mongo_db_major_version         = var.mongodb_version
  cluster_type                   = var.cluster_type
  termination_protection_enabled = var.termination_protection_enabled
  num_shards                     = var.replication_specs.num_shards
  paused                         = var.paused
  disk_size_gb                   = var.disk_size_gb
  provider_volume_type           = var.provider_volume_type
  provider_disk_iops             = var.provider_disk_iops
  redact_client_log_data         = true
  provider_name                  = var.provider_name
  provider_instance_size_name    = var.provider_instance_size_name
  encryption_at_rest_provider    = var.encryption_at_rest_provider

  replication_specs {
    num_shards = var.replication_specs.num_shards
    dynamic "regions_config" {
      for_each = var.replication_specs.regions_config
      content {
        region_name     = regions_config.value.region_name
        electable_nodes = regions_config.value.electable_nodes
        analytics_nodes = regions_config.value.analytics_nodes
        priority        = regions_config.value.priority
        read_only_nodes = regions_config.value.read_only_nodes
      }
    }
  }

  advanced_configuration {
    oplog_size_mb                      = var.oplog_size_mb
    transaction_lifetime_limit_seconds = var.transaction_lifetime_limit_seconds
    minimum_enabled_tls_protocol       = "TLS1_2"
    javascript_enabled                 = false
    tls_cipher_config_mode             = "CUSTOM"
    custom_openssl_cipher_config_tls12 = ["TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]
  }

  labels {
    key   = "label1"
    value = var.label1val
  }
  dynamic "labels" {
    for_each = local.labels
    content {
      key   = labels.key
      value = labels.value
    }
  }

  tags {
    key   = "tag1"
    value = var.tag1val
  }
  dynamic "tags" {
    for_each = local.tags
    content {
      key   = tags.key
      value = replace(tags.value, "/", "_")
    }
  }
}
