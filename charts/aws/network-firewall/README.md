# network-firewall

An AWS Network Firewall firewall and firewall policy, with its logging
configuration and RAM sharing, as AWS Controllers for Kubernetes (ACK)
objects. A port of terraform-aws-modules'
[`terraform-aws-network-firewall`](https://github.com/terraform-aws-modules/terraform-aws-network-firewall/tree/da5ce58c76ab0827c712ec881dad56b0e24c1114)
at v2.3.0 (`da5ce58c`), Apache-2.0; modified (a translation of the module's
variables and logic into a Helm chart). Bundt Package `aws-network-firewall`
(`package.yaml`). Checked against ACK's networkfirewall-controller v1.8.0
(pinned in `UPSTREAM.md`, "ACK controllers"). The module's rule-group
submodule is a separate module and is not part of this chart.

## What it renders

| Module resource | When | Object (`hfl.dev/id`) |
| --- | --- | --- |
| `aws_networkfirewall_firewall_policy` | `create` and `create_policy` | `FirewallPolicy` (`policy`) |
| `aws_networkfirewall_firewall` | `create` | `Firewall` (`firewall`), pointing at the policy with `firewallPolicyRef` (or `firewall_policy_arn`) |
| `aws_networkfirewall_logging_configuration` | `create_logging_configuration` | `Firewall` `spec.loggingConfiguration` |
| `aws_ram_resource_association` | `policy_ram_resource_associations`, `allow_alpha: true` | `ram.cloudcontrol.aws.hfl.dev` `ResourceAssociation` (`ram-association-<key>`) |

## Differences from Terraform

- Refused by name (ACK's `Firewall` and `FirewallPolicy` have no field):
  `availability_zone_change_protection`, `availability_zone_mapping`,
  `enabled_analysis_types`, `transit_gateway_id` (only a VPC firewall),
  `enable_monitoring_dashboard`, `policy_enable_tls_session_holding`,
  `policy_stateful_engine_options.flow_timeouts` and
  `policy_stateful_rule_group_reference.*.deep_threat_inspection`.
- The policy's resource policy (`aws_networkfirewall_resource_policy`) has no
  kind in ACK or ack-extensions: `create_policy_resource_policy`,
  `policy_attach_resource_policy`, `policy_resource_policy` and the actions
  and principals are refused.
- `policy_ram_resource_associations` names the policy by an ARN built from
  `account_id`, `region` and the policy name, since the RAM kind has no
  reference field. A value is the resource share ARN (a string) or
  `{resource_share_arn}`: the module's `map(string)` type reads
  `.resource_share_arn` of each value, which only an object has.
- `subnet_mapping` is required: ACK's `Firewall` places the firewall by its
  subnets (`subnetMappings`). `vpc_id` is sent with it.
- A stateless custom action's `dimension` (a string, or a list of strings or
  `{value}`) becomes `publishMetricAction.dimensions`.
