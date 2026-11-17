# lambda

An AWS Lambda function with its IAM execution role and policies, CloudWatch
log group, layer version, function URL, event source mappings, and (alpha)
resource-based permissions, as AWS Controllers for Kubernetes (ACK) objects.

Ported from
[`terraform-aws-modules/lambda`](https://github.com/terraform-aws-modules/terraform-aws-lambda)
(`v8.13.0`).

## What is ported

- **Function (`Function`):** `function_name`, `handler`, `runtime`,
  `package_type`, `image_uri` or `s3_existing_package`, `description`,
  `architectures`, `layers`, `kms_key_arn`, `code_signing_config_arn`,
  `memory_size`, `timeout`, `ephemeral_storage_size`, `publish`,
  `reserved_concurrent_executions`, `dead_letter_target_arn`,
  `environment_variables`, `tracing_mode`, `vpc_subnet_ids`,
  `vpc_security_group_ids`, `ipv6_allowed_for_dual_stack`, `image_config_*`,
  `snap_start`, `file_system_*`, `logging_*`, `create_async_event_config`,
  `durable_config_*`, `tenant_isolation_mode`, `function_tags`, `tags`.
- **IAM Role (`Role`):** `create_role`, `role_*`, `trusted_entities`,
  `attach_cloudwatch_logs_policy`, `attach_network_policy`,
  `attach_tracing_policy`, `attach_policy*`, `policy*`.
- **LogGroup (`LogGroup`):** `use_existing_cloudwatch_log_group`,
  `cloudwatch_logs_retention_in_days`, `cloudwatch_logs_kms_key_id`,
  `cloudwatch_logs_tags`.
- **LayerVersion (`LayerVersion`):** `create_layer`, `layer_name`,
  `compatible_runtimes`, `compatible_architectures`, `license_info`,
  `s3_existing_package`.
- **FunctionURLConfig (`FunctionURLConfig`):** `create_lambda_function_url`,
  `authorization_type`, `cors`, `invoke_mode`.
- **EventSourceMapping (`EventSourceMapping`):** `event_source_mapping`.
- **Permission (`Permission`, `lambda.aws.hfl.dev/v1alpha1`):**
  `allowed_triggers` (requires `allow_alpha: true`).

## What is refused

- Local zip / Docker packaging and S3 upload (`source_path`,
  `local_existing_package`, `build_in_docker`, `docker_*`, `store_on_s3`,
  `s3_bucket`, `s3_prefix`, `artifacts_dir`, etc.): pass `image_uri` or
  `s3_existing_package`.
- `cloudwatch_logs_log_group_class`, `cloudwatch_logs_deletion_protection_enabled`,
  `recursive_loop`, `replace_security_groups_on_destroy`,
  `managed_instances_*`: no ACK field.
- `skip_destroy`, `layer_skip_destroy`, `cloudwatch_logs_skip_destroy`,
  `timeouts`: Bundt `deletion_policy: ABANDON` and ACK retry replace them.
