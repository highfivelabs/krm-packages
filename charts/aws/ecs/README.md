# ecs

An Amazon ECS cluster with capacity providers, CloudWatch log group, task
execution and managed-instance IAM roles, security group, and services with
task definitions and application auto scaling, as AWS Controllers for
Kubernetes (ACK) objects.

Ported from
[`terraform-aws-modules/ecs`](https://github.com/terraform-aws-modules/terraform-aws-ecs)
(`v7.6.1`).

## What is ported

- **Cluster (`Cluster`, `CapacityProvider`):** `cluster_name`,
  `cluster_configuration`, `cluster_service_connect_defaults`,
  `cluster_setting`, `cluster_capacity_providers`, `capacity_providers`,
  `default_capacity_provider_strategy`, `cluster_tags`, `tags`.
- **Log group (`LogGroup`):** `create_cloudwatch_log_group`,
  `cloudwatch_log_group_name`, `cloudwatch_log_group_retention_in_days`,
  `cloudwatch_log_group_kms_key_id`, `cloudwatch_log_group_tags`.
- **IAM roles (`Role`, `InstanceProfile`):** `create_task_exec_iam_role`
  and its policy/statements/SSM/Secrets Manager ARNs;
  `create_infrastructure_iam_role` and `create_node_iam_instance_profile`
  when a capacity provider enables `managed_instances_provider`.
- **Security group (`SecurityGroup`):** `create_security_group`, `vpc_id`,
  `security_group_*` when managed instances are enabled.
- **Services (`TaskDefinition`, `Service`, `ScalableTarget`,
  `ScalingPolicy`, `ScheduledAction`):** each entry in `services` renders a
  `TaskDefinition`, `Service`, optional per-service task/exec IAM roles, and
  optional application auto scaling (`ScheduledAction` requires
  `allow_alpha: true`).

## What is refused

- `cloudwatch_log_group_class`: ACK's `LogGroup` (`cloudwatchlogs-controller`
  `v1.10.0`) has no `logGroupClass` field.
- `infrastructure_iam_role_source_policy_documents`,
  `infrastructure_iam_role_override_policy_documents`,
  `node_iam_role_source_policy_documents`,
  `node_iam_role_override_policy_documents`: merging arbitrary IAM policy
  documents by `sid` is not ported; use `*_statements`.
- `*_use_name_prefix`: accepted in part (the default `true` names the
  resource as given without a random suffix; `namePrefix` keeps two
  Instances' resources apart).
