# rds-proxy

An RDS Proxy with its target, endpoints, CloudWatch log group and IAM role, as
AWS Controllers for Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-rds-proxy`](https://github.com/terraform-aws-modules/terraform-aws-rds-proxy/tree/2524d394b402bf8b9deaa541231eba50d6745f2b)
at v4.4.1 (`2524d394`), Apache-2.0; modified (a translation of the module's
variables and logic into a Helm chart). Bundt Package `aws-rds-proxy`
(`package.yaml`). Checked against ACK's rds-controller v1.15.1, the first
release with `DBProxy` (`UPSTREAM.md`, "ACK controllers": the pin moved from
v1.14.2 for this chart), iam-controller v1.11.3 and cloudwatchlogs-controller
v1.10.0.

## What it renders

| Module resource | When | Object (`hfl.dev/id`) |
| --- | --- | --- |
| `aws_db_proxy` | `create` | `rds` `DBProxy` (`proxy`) |
| `aws_cloudwatch_log_group` | `manage_log_group` | `cloudwatchlogs` `LogGroup` (`log-group`) |
| `aws_iam_role`, `aws_iam_role_policy` | `create_iam_role` (and `create_iam_policy`) | `iam` `Role` (`role`) with the secrets policy as an inline policy |
| `aws_db_proxy_target` | `target_db_instance` or `target_db_cluster`, `allow_alpha: true` | `rds.aws.hfl.dev` `DBProxyTarget` (`target`) |
| `aws_db_proxy_endpoint` | `endpoints`, `allow_alpha: true` | `rds.cloudcontrol.aws.hfl.dev` `DBProxyEndpoint` (`endpoint-<key>`) |

## Differences from Terraform

- The default target group's connection pool (`connection_borrow_timeout`,
  `init_query`, `session_pinning_filters`, `max_connections_percent`,
  `max_idle_connections_percent`) has no field on ACK's `DBProxy` and no
  kind in ack-extensions. The first three are refused; the two percentages
  are accepted at the module's defaults only (90 and 50) and not sent, so
  AWS's own pool defaults apply (100 and 50).
- Refused by name (no `DBProxy` field): `default_auth_scheme`,
  `endpoint_network_type`, `target_connection_network_type`.
- `use_role_name_prefix` and `use_policy_name_prefix` are refused (ACK's
  `Role` takes a name; set `iam_role_name`); `iam_role_force_detach_policies`
  is accepted at true only (ACK always deletes inline policies with the
  role); `log_group_class` is refused (no field on ACK's `LogGroup`).
- `region` is required when the role's secrets policy is created: its
  `kms:ViaService` condition names `secretsmanager.<region>`, which
  Terraform reads from the provider.
