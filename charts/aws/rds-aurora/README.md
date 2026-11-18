# rds-aurora

An Amazon Aurora cluster (or a Multi-AZ DB cluster) with its instances,
custom endpoints, subnet, parameter and security groups, log groups,
enhanced monitoring role, read replica autoscaling and role associations,
as AWS Controllers for Kubernetes (ACK) objects. A port of
terraform-aws-modules'
[`terraform-aws-rds-aurora`](https://github.com/terraform-aws-modules/terraform-aws-rds-aurora/tree/652c0648a304e04bb53b5411874f519ace3585b6)
at v10.4.2 (`652c0648`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-rds-aurora`
(`package.yaml`). Checked against ACK's rds-controller v1.15.1,
ec2-controller v1.23.1, iam-controller v1.11.3, cloudwatchlogs-controller
v1.10.0 and applicationautoscaling-controller v1.9.0 (`UPSTREAM.md`, "ACK
controllers"). The module's `dsql` submodule is not ported.

## Example

```yaml
name: orders
engine: aurora-postgresql
engine_version: "16.4"
cluster_instance_class: db.r8g.large
instances: {one: {}, two: {}}
vpc_id: vpc-0123456789abcdef0
create_db_subnet_group: true
subnets: [subnet-0123456789abcdef0, subnet-0123456789abcdef1]
security_group_ingress_rules:
  vpc: {cidr_ipv4: 10.0.0.0/16}
enabled_cloudwatch_logs_exports: [postgresql]
create_cloudwatch_log_group: true
```

renders a `DBCluster` (`hfl.dev/id` `cluster`), two `DBInstance`s
(`instance-one`, `instance-two`, identifiers `orders-one` and
`orders-two`), the `DBSubnetGroup` (`db-subnet-group`), the
`SecurityGroup` (`security-group`, port 5432 from the VPC) and the log
group `/aws/rds/cluster/orders/postgresql` (`log-group-postgresql`). With
`skip_final_snapshot` false (the default) the cluster leaves a final
snapshot named `final_snapshot_identifier`, or `<identifier>-final` when
that is unset (Terraform would fail on destroy).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_rds_cluster.this` | `rds/DBCluster`: every argument the module sets that ACK's kind has; the subnet, cluster parameter and security groups and the monitoring role by reference (`dbSubnetGroupRef`, `dbClusterParameterGroupRef`, `vpcSecurityGroupRefs`, `monitoringRoleRef`) or by name and ID; `master_password_wo` as a Kubernetes Secret reference (`masterUserPassword`); a point-in-time restore through `sourceDBClusterIdentifier` and its fields; the final snapshot and automated backups through rds-controller's deletion annotations |
| `aws_rds_cluster_instance.this` | `rds/DBInstance` with `dbClusterIdentifierRef` (ACK makes it once the cluster exists). The cluster's values the module coalesces into each instance (class, CA, monitoring, Performance Insights, maintenance window) are carried the same way. The instance takes its engine version and subnet group from the cluster |
| `aws_rds_cluster_endpoint.this` | `rds/DBClusterEndpoint` with `dbClusterIdentifierRef` |
| `aws_rds_cluster_role_association.this` | ack-extensions' `DBRoleAssociation` with `dbClusterRef` (`allow_alpha: true`) |
| `aws_db_subnet_group.this` | `rds/DBSubnetGroup` |
| `aws_rds_cluster_parameter_group.this`, `aws_db_parameter_group.this` | `rds/DBClusterParameterGroup`, `rds/DBParameterGroup` (`parameterOverrides`) |
| `aws_security_group.this` and its rules | `ec2/SecurityGroup` with `ingressRules` and `egressRules` inline, ports defaulting to the cluster's as the module's do; `self` is the group itself |
| `aws_iam_role.rds_enhanced_monitoring` and its attachment | `iam/Role` with `AmazonRDSEnhancedMonitoringRole` in `policies` |
| `aws_appautoscaling_target.this`, `aws_appautoscaling_policy.this` | `applicationautoscaling/ScalableTarget` and a target tracking `ScalingPolicy` (`resourceRef` to the target) on `rds:cluster:ReadReplicaCount` |
| `aws_cloudwatch_log_group.this` | `cloudwatchlogs/LogGroup`, `/aws/rds/cluster/<identifier>/<log>`; `cloudwatch_log_group_skip_destroy` keeps it on delete |

A serverless (v1) cluster has no instances, endpoints or autoscaling, as in
the module.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`cluster`, `dbSubnetGroup`, `clusterParameterGroup`,
`dbParameterGroup`, `securityGroup`, `monitoringRole`), `namePrefix` (on
every name the chart gives; instances, endpoints and log groups follow the
cluster's identifier), `labels` (Kubernetes only), `allow_alpha` and
`deletion_policy`; `tags` is the module's own. `name` and `engine` are
required (ACK's `DBCluster` requires an engine; Terraform's provider takes
`aurora`), and `vpc_id` with `create_security_group: true` (the default).
Identifiers are lower-cased, as RDS stores them.

Refused (`upstream.yaml`):

- `apply_immediately: false`, here or on an instance: ACK applies every
  change immediately.
- `cluster_use_name_prefix`, `instances_use_identifier_prefix`, an
  instance's `identifier_prefix`, `iam_role_use_name_prefix`: Terraform's
  random suffix would change between renders. `security_group_use_name_prefix`
  and the parameter groups' `use_name_prefix` (all `true` by default) name
  the groups as given.
- `allow_major_version_upgrade`, `cluster_db_instance_parameter_group_name`,
  `enable_local_write_forwarding`, `engine_lifecycle_support`,
  `cluster_members`: no field on ACK's `DBCluster`.
- `cluster_scalability_type` and `shard_group`: `DBCluster` cannot make an
  Aurora Limitless cluster, so ack-extensions' `DBShardGroup` has nothing to
  join.
- `cluster_ca_cert_identifier` without instances (a Multi-AZ DB cluster):
  `DBCluster` has no CA field; with instances it is carried to each.
- `s3_import`: no restore from S3. `restore_to_point_in_time`'s
  `source_cluster_resource_id`: no field.
- `master_password_wo` as a string: the password is never a value. Set it
  to `{secretName, key}`, a Kubernetes Secret in the component's namespace
  (`docs/secrets.md`); it is used when `manage_master_user_password` is
  false. `master_password_wo_version`: ACK sends the Secret's value when
  the reference changes; point it at a new Secret or key instead.
- `vpc_security_group_ids` with the chart's own group: `DBCluster` cannot
  take references and IDs together.
- Per-rule `name`, `tags` and a `region` other than the chart's on
  security group rules: ACK keeps rules inline. A parameter's
  `apply_method`: ACK picks immediate or pending-reboot from the
  parameter's metadata.
- `cloudwatch_log_group_class`: no field on ACK's `LogGroup`.
- `cluster_activity_stream`: no kind starts one.
- The five master user password rotation variables: no rotation kind
  (secretsmanager-controller pull request #69).
- `cluster_timeouts`, `instance_timeouts`: ACK retries until synced, and
  Bundt waits.

## Outputs

`db_subnet_group_name`, `cluster_arn`, `cluster_id`, `cluster_resource_id`,
`cluster_members`, `cluster_endpoint`, `cluster_reader_endpoint`,
`cluster_engine_version_actual` (the spec's version, which ACK fills in
from RDS), `cluster_database_name`, `cluster_port`,
`cluster_master_username`, `cluster_master_user_secret`,
`cluster_hosted_zone_id`, `cluster_instances` (each instance's identifier,
ARN, endpoint and class by key), `additional_cluster_endpoints`,
`cluster_role_associations`, `enhanced_monitoring_iam_role_name`,
`_arn` and `_unique_id`, `security_group_id`,
`db_cluster_parameter_group_arn` and `_id`, `db_parameter_group_arn` and
`_id`, `db_cluster_cloudwatch_log_groups`. Not provided: the cluster's CA
certificate and its validity and its upgrade rollout order (ACK's
`DBCluster` neither takes nor reports them), and the activity stream's,
rotation's and shard group's (refused).

## Tests

`task test CHART=aws/rds-aurora`: golden renders (the module README's
usage, a full case, serverless v1, a Multi-AZ DB cluster restored to a
point in time, every name) validated against the pinned controllers' CRDs,
and the refusal cases. `tests/mock.values.yaml` is the case the fake-AWS
run applies, in the VPC and subnets `tests/mock.needs.yaml` makes
(`docs/aws.md`, section 8).
