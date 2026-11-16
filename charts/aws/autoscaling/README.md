# autoscaling

An Amazon EC2 Auto Scaling group with its launch template, IAM role and
instance profile, lifecycle hooks and traffic sources, and its scaling
policies, scheduled actions and warm pool, as AWS Controllers for
Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-autoscaling`](https://github.com/terraform-aws-modules/terraform-aws-autoscaling/tree/f90a581bcca0bab571022a021db155f2860e9b32)
at v9.3.2 (`f90a581b`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-autoscaling`
(`package.yaml`). Checked against ACK's autoscaling-controller v1.6.0,
ec2-controller v1.23.1 and iam-controller v1.11.3 (`UPSTREAM.md`, "ACK
controllers"), and ack-extensions' `ScalingPolicy`, `ScheduledAction` and
`WarmPool` (`crds/ack-extensions`).

## Example

```yaml
name: web
min_size: 1
max_size: 3
vpc_zone_identifier: [subnet-0123456789abcdef0, subnet-0123456789abcdef1]
image_id: ami-0123456789abcdef0
instance_type: t3.micro
security_groups: [sg-0123456789abcdef0]
create_iam_instance_profile: true
iam_role_name: web
iam_role_policies:
  ssm: arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore
allow_alpha: true
scaling_policies:
  cpu:
    policy_type: TargetTrackingScaling
    target_tracking_configuration:
      predefined_metric_specification: {predefined_metric_type: ASGAverageCPUUtilization}
      target_value: 50
tags: {team: web}
```

renders an `AutoScalingGroup` (`hfl.dev/id` `autoscaling-group`), its
`LaunchTemplate` (`launch-template`), a `Role` (`iam-role`) and
`InstanceProfile` (`iam-instance-profile`), and a `ScalingPolicy`
(`scaling-policy-cpu`) (`tests/golden/full.yaml` has every kind).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_autoscaling_group.this` (or `.idc`) | `autoscaling/AutoScalingGroup`: sizes, Availability Zones or `vpcZoneIdentifier` (the subnets, comma-separated), the Availability Zone distribution, capacity rebalance, cooldown, warmup, health checks, `lifecycleHookSpecificationList` (`initial_lifecycle_hooks`), the instance lifecycle and maintenance policies, `launchTemplate` or `mixedInstancesPolicy` (instances distribution, overrides with instance requirements), maximum instance lifetime, scale-in protection, placement group, service-linked role, termination policies, `trafficSources`, `tags` |
| `aws_launch_template.this` | `ec2/LaunchTemplate`: every argument the module passes, in `data` (ACK's `RequestLaunchTemplateData`); `description` is the version's `versionDescription`; `default_version` is `defaultVersion` |
| `aws_autoscaling_traffic_source_attachment.this` | the group's `trafficSources` |
| `aws_autoscaling_schedule.this` | `autoscaling.aws.hfl.dev/ScheduledAction` (ack-extensions), one per key, named by its key |
| `aws_autoscaling_policy.this` | `autoscaling.aws.hfl.dev/ScalingPolicy` (ack-extensions), one per key, named `name` else its key: simple, step, target tracking (predefined or customized metrics, metric math) and predictive |
| the group's `warm_pool` block | `autoscaling.cloudcontrol.aws.hfl.dev/WarmPool` (ack-extensions, Cloud Control): ACK's `AutoScalingGroup` has no warm pool |
| `aws_iam_role.this`, `aws_iam_role_policy_attachment.this` | `iam/Role` trusted by `ec2.amazonaws.com` (`ec2.amazonaws.com.cn` in China), the managed policies in `policies` |
| `aws_iam_instance_profile.this` | `iam/InstanceProfile` (`roleRef`) |

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`autoScalingGroup`, `launchTemplate`, `iamRole`,
`instanceProfile`), `namePrefix` (on those four names and the `Name` tag),
`labels` (Kubernetes only), `allow_alpha` and `deletion_policy`; `tags` is
the module's own. `name`, `min_size` and `max_size` are required. `region`
is the annotation `services.k8s.aws/region` on every object.

**Scaling policies, schedules and the warm pool need `allow_alpha: true`.**
ACK's autoscaling-controller v1.6.0 has only `AutoScalingGroup`; they are
ack-extensions kinds, outside ACK's releases (`docs/design.md`, "Alpha
kinds"). Without policies the group does not scale on load.

**The launch template is named, not referenced.** ACK's
`AutoScalingGroup` takes `launchTemplate.launchTemplateName` and a
version, with no ID field and no reference to a `LaunchTemplate` object;
ACK retries the group until the template exists. The chart's own template
is used at `$Latest` unless `launch_template_version` is set (the module
uses its `latest_version`, the same version for every launch). An
existing template takes `create_launch_template: false` and
`launch_template_name` (its name, not its ID).

**Tags reach the instances through the launch template.** ACK's
`AutoScalingGroup` sends every tag with `PropagateAtLaunch: false`. The
module propagates all of the group's tags (`tags`, `Name` = `instance_name`
or `name`, `autoscaling_group_tags`) except
`autoscaling_group_tags_not_propagate_at_launch`, so the chart adds those
tags to its launch template's `instance` tag specification (after the
module's own `tag_specifications`; the group's value wins on a shared
key). With `create_launch_template: false` the instances do not get them.

**The instance profile is named.** The module gives the launch template
the profile's ARN, which is only in the `InstanceProfile`'s status; the
chart gives its name, which EC2 resolves the same way. As in the module,
the role is named `iam_instance_profile_name` else `iam_role_name`, and
the profile `iam_role_name` (else the role's name, where Terraform would
generate one).

**Set once at creation**: `initial_lifecycle_hooks` and
`traffic_source_attachments` are fields of `CreateAutoScalingGroup` only,
and ACK neither updates nor attaches them later. Terraform's
`initial_lifecycle_hook` is the same; a traffic source added later
needs the group recreated.

**Desired capacity.** Unset, AWS starts the group at `min_size` and scaling
moves it. Set, ACK writes it back on every change it sees, so with scaling
policies leave it unset. `ignore_desired_capacity_changes: true` is
accepted only without `desired_capacity`.

**Deleting** the group force-deletes it (ACK's controller always sends
`ForceDelete`), whatever `force_delete` says.

Refused (`upstream.yaml`):

- `launch_template_id`, and an override's
  `launch_template_specification.launch_template_id`: ACK names launch
  templates; set `launch_template_name`.
- `enabled_metrics`, `metrics_granularity`, `suspended_processes`,
  `instance_refresh`, `context`: no field on ACK's `AutoScalingGroup` and
  no ack-extensions kind (`docs/aws.md`, gaps).
- `force_delete_warm_pool`: no field on ack-extensions' `WarmPool`.
- `wait_for_capacity_timeout`, `wait_for_elb_capacity`,
  `min_elb_capacity`, `ignore_failed_scaling_activities`, `timeouts`:
  Terraform's waits; Bundt reads the objects' conditions.
- `update_default_version: true`: ACK's `LaunchTemplate` makes a new
  version on each change and moves the default only to `default_version`.
- `block_device_mappings[*].ebs.volume_initialization_rate`,
  `network_interfaces[*].connection_tracking_specification` and
  `.ena_srd_specification`, `network_performance_options`,
  `secondary_interfaces`: no field on ACK's `LaunchTemplate`.
- `ignore_desired_capacity_changes: true` with `desired_capacity`: above.
- `use_name_prefix`, `launch_template_use_name_prefix`,
  `iam_role_use_name_prefix` (all `true` by default) name the group, the
  template and the role as given, without Terraform's random suffix; the
  `namePrefix` value keeps two Instances apart.

The module's launch template leaves `instance_requirements.network_bandwidth_gbps`
out, and so does the chart (an override's is carried).

## Outputs

All 27 of the module's: the launch template's `id`, `arn`, `name`,
`latest_version` and `default_version`; the group's `id` and `name` (its
name), `arn`, sizes, cooldown, health check settings, Availability
Zones, `vpc_zone_identifier` (split back into a list), load balancers and
target group ARNs; `autoscaling_group_enabled_metrics` (always empty:
`enabled_metrics` is refused); `autoscaling_schedule_arns` and
`autoscaling_policy_arns` by key; the role's name, ARN and unique ID; the
instance profile's ARN (the given `iam_instance_profile_arn` when the
chart makes none), ID (its name) and unique ID.

## Tests

`task test CHART=aws/autoscaling`: golden renders validated against
autoscaling-controller v1.6.0's, ec2-controller v1.23.1's and
iam-controller v1.11.3's CRDs and ack-extensions' copied CRDs, and the
refusal cases. `tests/mock.values.yaml` is the case the fake-AWS run
applies (an empty group, its template, role, profile, a lifecycle hook, a
scaling policy and a schedule), in the VPC and subnet
`tests/mock.needs.yaml` makes (`docs/aws.md`, "Charts").

## From Terraform

A `module "asg" { source = "terraform-aws-modules/autoscaling/aws" ... }`
block's arguments are this chart's values one to one (`task convert`),
except the refusals above.
