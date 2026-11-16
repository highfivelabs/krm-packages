# alb

An Elastic Load Balancing v2 load balancer (application, network or
gateway) with its listeners, listener rules, target groups and security
group, as AWS Controllers for Kubernetes (ACK) objects. A port of
terraform-aws-modules'
[`terraform-aws-alb`](https://github.com/terraform-aws-modules/terraform-aws-alb/tree/6c6e48c10d450d01cf6715346e926df0b858efc2)
at v10.5.1 (`6c6e48c1`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-alb`
(`package.yaml`). Checked against ACK's elbv2-controller v1.10.0 and
ec2-controller v1.23.1 (`UPSTREAM.md`, "ACK controllers").

## Example

```yaml
name: my-alb
vpc_id: vpc-abcde012
subnets: [subnet-abcde012, subnet-bcde012a]
enable_deletion_protection: false
security_group_ingress_rules:
  all_http: {from_port: 80, cidr_ipv4: 0.0.0.0/0}
listeners:
  http:
    port: 80
    protocol: HTTP
    forward: {target_group_key: app}
target_groups:
  app: {name: app, protocol: HTTP, port: 80, target_type: ip, create_attachment: false}
```

renders a `LoadBalancer` (`hfl.dev/id` `load-balancer`), its
`SecurityGroup` (`security-group`), a `Listener` per listener
(`listener-<key>`), a `Rule` per listener rule (`rule-<listener>/<rule>`)
and a `TargetGroup` per target group (`target-group-<key>`)
(`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_lb.this` | `LoadBalancer`: `name`, `type`, `scheme` (`internal`), `ipAddressType`, `customerOwnedIPv4Pool`, `subnets`, `subnetMappings`, `securityGroupRefs` (the chart's group) or `securityGroups`, `tags`, and `attributes` |
| `aws_security_group.this` and its rules | `SecurityGroup` with `ingressRules` and `egressRules` inline, as `charts/aws/security-group` |
| `aws_lb_listener.this` | `Listener` (`loadBalancerRef`): `port`, `protocol` (none for GENEVE), `sslPolicy` (the module's default for HTTPS and TLS), `alpnPolicy`, `certificates` (`certificate_arn`), `mutualAuthentication`, `defaultActions` |
| `aws_lb_listener_rule.this` | `Rule` (`listenerRef`): `priority`, `actions`, `conditions` |
| `aws_lb_target_group.this` and its attachments | `TargetGroup`: `name`, `targetType`, `port`, `protocol`, `protocolVersion`, `vpcID`, `ipAddressType`, the health check fields, `attributes`, and `targets` (the group's `target_id` and the `additional_target_group_attachments` for it) |

Load balancer attributes, each only for the types that have it, as the
Terraform provider sends them, and only when set (or defaulted by the
module): access, connection and health check logs, client keep-alive,
desync mitigation, DNS client routing (network), dropping invalid header
fields, cross-zone load balancing (network, gateway), deletion protection,
HTTP/2, the TLS headers, WAF fail-open, the XFF client port and processing
mode, zonal shift, idle timeout, preserving the host header. The module
defaults `enable_deletion_protection` to `true`: ACK cannot delete such a
load balancer until it is set to `false`.

Target group attributes: deregistration delay and connection termination,
Lambda multi-value headers, the load balancing algorithm and anomaly
mitigation, cross-zone, client IP preservation, proxy protocol v2, slow
start, stickiness (load balancer or application cookie), target failover,
target group health (DNS failover, unhealthy state routing) and target
health state.

Actions (a listener's default actions, a rule's actions): authenticate
with Cognito or OIDC, fixed response, forward (to `target_group_key`, by
`targetGroupRef`, or to `target_group_arn`), weighted forward with
stickiness, redirect. Conditions: host header, HTTP header, request
method, path pattern, query string, source IP.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`loadBalancer`, `securityGroup`), `namePrefix` (on the
load balancer's, its group's and each target group's name), `labels`
(Kubernetes only) and `deletion_policy`; `tags` is the module's own.
`name` (or `names.loadBalancer`) is required, and so is each target
group's `name`: Terraform's generated names would change between renders.
`vpc_id` is required with `create_security_group: true` (the default).

Refused (`upstream.yaml`):

- `route53_records`: an alias record needs the load balancer's DNS name and
  zone ID, which exist only in its status, and ACK's `RecordSet` has no
  reference to a `LoadBalancer`. Make the records in a `route53`
  component from this one's `dns_name` and `zone_id` outputs.
- `associate_web_acl`, `web_acl_arn`: ack-extensions' `WebACLAssociation`
  takes the load balancer's ARN only as a string.
- `target_groups.*.attach_lambda_permission`: likewise for ack-extensions'
  Lambda `Permission` and the target group's ARN.
- The listener attributes (`routing_http_*`, `tcp_idle_timeout_seconds`),
  `additional_certificate_arns`,
  `mutual_authentication.advertise_trust_store_ca_names`, `jwt_validation`
  actions, rule `transform`s and `regex_values` conditions: no field on
  elbv2-controller v1.10.0's kinds. A rule needs `priority`.
- `security_groups` with `create_security_group: true`: ACK's
  `LoadBalancer` cannot take group references and IDs together.
- `enforce_security_group_inbound_rules_on_private_link_traffic`,
  `ipam_pools`, `minimum_load_balancer_capacity`: no field on ACK's
  `LoadBalancer`.
- `name_prefix`, `target_groups.*.name_prefix`: Terraform's random suffix.
- `timeouts`: ACK retries until synced, and Bundt waits.

## Outputs

`id` and `arn` (the load balancer's ARN), `arn_suffix`, `dns_name`,
`zone_id`, `listeners`, `listener_rules` and `target_groups` (each one's
ARN by key), `security_group_id`, `security_group_arn` (built as the
provider builds it). Not provided: `route53_records` (refused).

## Tests

`task test CHART=aws/alb`: golden renders validated against
elbv2-controller v1.10.0's and ec2-controller v1.23.1's CRDs, and the
refusal cases. `tests/mock.values.yaml` is the case the fake-AWS run
applies, in the VPC and subnets `tests/mock.needs.yaml` makes
(`docs/aws.md`, "Charts").
