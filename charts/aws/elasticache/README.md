# elasticache

An Amazon ElastiCache replication group or cache cluster (Valkey, Redis
OSS or Memcached) with its parameter group, subnet group, security group
and log groups, as AWS Controllers for Kubernetes (ACK) objects. A port of
terraform-aws-modules'
[`terraform-aws-elasticache`](https://github.com/terraform-aws-modules/terraform-aws-elasticache/tree/41a277b73b9d375a2856886ab8469fd3f8b6d0ef)
at v1.11.2 (`41a277b7`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-elasticache`
(`package.yaml`). Checked against ACK's elasticache-controller v1.12.0,
ec2-controller v1.23.1 and cloudwatchlogs-controller v1.10.0
(`UPSTREAM.md`, "ACK controllers"). The module's `user-group` and
`serverless-cache` submodules are not ported.

## Example

```yaml
replication_group_id: sessions
node_type: cache.t4g.small
engine_version: "7.1"
num_cache_clusters: 2
subnet_ids: [subnet-0123456789abcdef0, subnet-0123456789abcdef1]
vpc_id: vpc-0123456789abcdef0
security_group_rules:
  vpc: {cidr_ipv4: 10.0.0.0/16, description: Redis from the VPC}
```

renders a `ReplicationGroup` (`hfl.dev/id` `replication-group`), its
`CacheSubnetGroup` (`subnet-group`), its `SecurityGroup`
(`security-group`) and the slow log's `LogGroup` (`log-group-slow-log`)
(`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_elasticache_replication_group.this` | `ReplicationGroup`: every argument the module sets that ACK's kind has, the groups by reference (`cacheParameterGroupRef`, `cacheSubnetGroupRef`, `securityGroupRefs`) or by name and ID, `logDeliveryConfigurations`. `num_cache_clusters` (cluster mode disabled) becomes one node group with `num_cache_clusters - 1` replicas, as ACK's kind has no `numCacheClusters` |
| `aws_elasticache_cluster.this` | `CacheCluster`: alone, or in a replication group (`replication_group_id`; the chart's own by `replicationGroupRef`), carrying only what a member can set, as the module does |
| `aws_elasticache_parameter_group.this` | `CacheParameterGroup` (`cluster-enabled: yes` added with `cluster_mode_enabled`, as the module adds it) |
| `aws_elasticache_subnet_group.this` | `CacheSubnetGroup` |
| `aws_security_group.this` and its rules | `SecurityGroup` with `ingressRules` and `egressRules` inline (an ingress rule's ports default to the engine's, as the module's) |
| `aws_cloudwatch_log_group.this` | `cloudwatchlogs/LogGroup`, `/aws/elasticache/<name>`, 14 days unless set; `cloudwatch_log_group_skip_destroy` keeps it on delete |

Every object carries the module's tags, `tags` with
`terraform-aws-modules = elasticache`.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`replicationGroup`, `cluster`, `parameterGroup`,
`subnetGroup`, `securityGroup`), `namePrefix` (on every name the chart
gives), `labels` (Kubernetes only) and `deletion_policy`; `tags` is the
module's own. IDs and group names are lower-cased, as ElastiCache stores
them. `replication_group_id` (with `create_replication_group: true`, the
default), `node_type`, `subnet_ids` (with `create_subnet_group: true`)
and `vpc_id` (with `create_security_group: true`) are required. A
parameter group the chart makes without `parameter_group_name` is named
`<cluster_id or replication_group_id>-<family>`, without the module's
random suffix. `log_delivery_configuration` is a map Helm merges with its
default: set `slow-log: null` to drop the default slow log.

The replication group and cluster name their log groups by name, with no
reference: ACK retries until the `LogGroup` exists.

Refused (`upstream.yaml`):

- `create_primary_global_replication_group`,
  `create_secondary_global_replication_group`,
  `global_replication_group_id`, `global_replication_group_id_suffix`: no
  `GlobalReplicationGroup` kind (ACK's elasticache-controller pull request
  #237), and no field on `ReplicationGroup` to join one.
- `auth_token`, `auth_token_update_strategy`: a password is never a value;
  ACK takes the token from a Kubernetes Secret, which the chart does not
  make yet. `user_group_ids` (RBAC) is carried.
- `apply_immediately: false`: ACK applies every change immediately.
- `final_snapshot_identifier`, `transit_encryption_mode`, `cluster_mode`:
  no field on ACK's kinds; `cluster_mode_enabled` is carried.
- `log_delivery_configuration.*.cloudwatch_log_group_class`: no field on
  ACK's `LogGroup`.
- `security_group_ids` with `create_security_group: true`: ACK's kinds
  cannot take group references and IDs together.
- `security_group_rules.*.tags` are not carried (ACK keeps rules inline,
  with no per-rule tags); `security_group_use_name_prefix: true` (the
  default) names the group as given, without Terraform's random suffix.
- `timeouts`: ACK retries until synced, and Bundt waits.

## Outputs

`cluster_arn`, `cluster_cache_nodes` (each node's id, address, port and
Availability Zone), `cluster_address`, `cluster_configuration_endpoint`,
`replication_group_arn`, `replication_group_id`,
`replication_group_configuration_endpoint_address`,
`replication_group_member_clusters`,
`replication_group_primary_endpoint_address`,
`replication_group_reader_endpoint_address`, `replication_group_port`,
`cloudwatch_log_groups` (each one's name by key), `parameter_group_arn`,
`parameter_group_id`, `subnet_group_name`, `security_group_id`,
`security_group_arn` (built as the provider builds it). Not provided: the
`*_engine_version_actual` outputs (ACK reports no running version), the
global replication group's (refused), and `cloudwatch_log_group_name` and
`cloudwatch_log_group_arn` (the module reads its log group map by index 0,
so they are always null).

## Tests

`task test CHART=aws/elasticache`: golden renders validated against the
pinned controllers' CRDs, and the refusal cases. `tests/mock.values.yaml`
is the case the fake-AWS run applies, in the VPC and subnets
`tests/mock.needs.yaml` makes (`docs/aws.md`, "Charts").

## From Terraform

A `module "elasticache" { source = "terraform-aws-modules/elasticache/aws" ... }`
block's arguments are this chart's values one to one (`task convert`),
except the refusals above.
