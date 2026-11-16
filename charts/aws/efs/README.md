# efs

An Amazon EFS file system with its policy, backup policy, replication,
mount targets, their security group and access points, as AWS Controllers
for Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-efs`](https://github.com/terraform-aws-modules/terraform-aws-efs/tree/e0ec33d8436363960395b859aff0168ce25c27a4)
at v2.2.1 (`e0ec33d8`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-efs`
(`package.yaml`). Checked against ACK's efs-controller v1.8.0 and
ec2-controller v1.23.1 (`UPSTREAM.md`, "ACK controllers").

## Example

```yaml
name: shared
security_group_vpc_id: vpc-0123456789abcdef0
deny_nonsecure_transport: false
deny_nonsecure_transport_via_mount_target: false
policy_statements:
  mount:
    actions: [elasticfilesystem:ClientMount]
    resources: ["*"]
    principals: [{type: AWS, identifiers: [arn:aws:iam::111122223333:role/app]}]
    condition: [{test: Bool, variable: aws:SecureTransport, values: ["true"]}]
mount_targets:
  a: {subnet_id: subnet-0123456789abcdef0}
security_group_ingress_rules:
  vpc: {cidr_ipv4: 10.0.0.0/16, description: NFS from the VPC}
access_points:
  app:
    posix_user: {gid: 1000, uid: 1000}
    root_directory: {path: /app, creation_info: {owner_gid: 1000, owner_uid: 1000, permissions: "755"}}
```

renders a `FileSystem` (`hfl.dev/id` `file-system`) with its policy and
backup policy, a `MountTarget` per mount target (`mount-target-<key>`),
their `SecurityGroup` (`security-group`) and an `AccessPoint` per access
point (`access-point-<key>`) (`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_efs_file_system.this` | `FileSystem`: `availabilityZoneName`, `performanceMode`, `encrypted`, `kmsKeyID`, `throughputMode`, `provisionedThroughputInMiBps`, `lifecyclePolicies` (one per transition, as the module writes them), `fileSystemProtection`, `tags` (`Name` is `name`) |
| `aws_efs_file_system_policy.this` | `FileSystem.policy`, built as `data.aws_iam_policy_document.policy` builds it: the source documents, the statements (by key, each `Sid` defaulting to its key), the override documents, merged by `Sid` |
| `aws_efs_backup_policy.this` | `FileSystem.backupPolicy.status` (`ENABLED` or `DISABLED`) |
| `aws_efs_replication_configuration.this` | `FileSystem.replicationConfiguration` (one destination) |
| `aws_efs_mount_target.this` | `MountTarget` (`fileSystemRef`): `subnetID`, `ipAddress`, `securityGroupRefs` (the chart's group) or `securityGroups` |
| `aws_security_group.this` and its rules | `SecurityGroup` with `ingressRules` and `egressRules` inline, as `charts/aws/security-group` (ports default to 2049, as the module's) |
| `aws_efs_access_point.this` | `AccessPoint` (`fileSystemRef`): `posixUser`, `rootDirectory`, `tags` (`Name` is the entry's name, else its key) |

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`securityGroup`), `namePrefix` (on the security
group's name and on the file system's and access points' `Name` tags),
`labels` (Kubernetes only) and `deletion_policy`; `tags` is the module's
own. With mount targets and `create_security_group: true` (the default),
`security_group_vpc_id` and a group name (`security_group_name`, else
`name`) are required. `region` is the annotation
`services.k8s.aws/region` on every object.

**The policy's defaults are refused.** `attach_policy`,
`deny_nonsecure_transport` and `deny_nonsecure_transport_via_mount_target`
default to `true`, and the module writes both transport statements, and
any statement without `resources`, with the file system's own ARN. That
ARN holds the ID AWS assigns, which exists only in the `FileSystem`'s
status, and ACK's `FileSystem` takes its policy as a string with no
reference to it. So set both transport flags to `false` and give each
statement its `resources` (a file system policy applies only to its file
system, so `"*"` is the usual choice), or set `attach_policy: false`.

Refused (`upstream.yaml`):

- `deny_nonsecure_transport`, `deny_nonsecure_transport_via_mount_target`
  (when `true` with `attach_policy`), and a `policy_statements` entry
  without `resources`: above.
- `creation_token`: ACK's `FileSystem` has no field; the controller sets
  its own.
- `bypass_policy_lockout_safety_check`: no field on ACK's `FileSystem`.
- `mount_targets.*.ip_address_type`, `mount_targets.*.ipv6_address`: no
  field on ACK's `MountTarget`; `mount_targets.*.security_groups` with
  `create_security_group: true`: ACK's `MountTarget` cannot take group
  references and IDs together.
- `security_group_*_rules.*.name` and `.tags` are not carried (ACK keeps
  rules inline, with no per-rule tags); `security_group_use_name_prefix:
  true` names the group as given, without Terraform's random suffix.
- `policy_statements.*.conditions` is not carried, as the module does not
  carry it (its dynamic block reads `condition`).

## Outputs

`arn`, `id`, `dns_name` (built as the provider builds it),
`size_in_bytes` (the metered size's value), `mount_targets` and
`access_points` (each one's ID by key), `security_group_id`,
`security_group_arn` (built as the provider builds it) and
`replication_configuration_destination_file_system_id`.

## Tests

`task test CHART=aws/efs`: golden renders validated against
efs-controller v1.8.0's and ec2-controller v1.23.1's CRDs, and the refusal
cases. `tests/mock.values.yaml` is the case the fake-AWS run applies, in
the VPC and subnet `tests/mock.needs.yaml` makes (`docs/aws.md`,
"Charts").

## From Terraform

A `module "efs" { source = "terraform-aws-modules/efs/aws" ... }` block's
arguments are this chart's values one to one (`task convert`), except the
refusals above.
