# eventbridge

An EventBridge bus with its rules and targets, archives, permissions, log
delivery, IAM roles and pipes, and, as alpha kinds, connections, API
destinations, schedule groups, schedules and a schemas discoverer, as AWS
Controllers for Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-eventbridge`](https://github.com/terraform-aws-modules/terraform-aws-eventbridge/tree/e29707a5057ec9b0f6ea87686e9a5b48b4addd11)
at v4.3.3 (`e29707a5`), Apache-2.0, Copyright 2021 Sven Lito; modified (a
translation of the module's variables into a Helm chart, and its pipes'
service integration policies, `iam_pipes.tf`, carried as data in
`templates/pipes.yaml`). Bundt Package `aws-eventbridge` (`package.yaml`).
Checked against ACK's eventbridge-controller v1.9.2, pipes-controller
v1.8.0, iam-controller v1.11.3 and cloudwatchlogs-controller v1.10.0
(`UPSTREAM.md`, "ACK controllers"), and ack-extensions at the catalog's
`ACKX_COMMIT` (`crds/ack-extensions`).

## Example

The module's README usage (`tests/basic.values.yaml`):

```yaml
account_id: "111122223333"
region: us-east-1
bus_name: my-bus
attach_sqs_policy: true
sqs_target_arns: ["arn:aws:sqs:us-east-1:111122223333:orders"]
rules:
  orders:
    description: Capture all order data
    event_pattern: '{"source":["myapp.orders"]}'
    enabled: true
targets:
  orders:
    - name: send-orders-to-sqs
      arn: arn:aws:sqs:us-east-1:111122223333:orders
      dead_letter_arn: arn:aws:sqs:us-east-1:111122223333:orders-dlq
      target_id: send-orders-to-sqs
tags: {Name: my-bus}
```

renders the `EventBus` (`hfl.dev/id` `bus`), the `Rule` `orders-rule`
(`rule-orders`) with its target inline, the module's `Role` (`role`) with
the SQS `Policy` (`policy-sqs`) attached by reference, and the bus's log
delivery source (`log-source`), which the module makes by default.

## What it renders

| Module resource | When | Object (`hfl.dev/id`) |
| --- | --- | --- |
| `aws_cloudwatch_event_bus` | `create_bus` | `eventbridge/EventBus` `bus`: `name` (`bus_name`), `eventSourceName`, `policy` (the permissions), `tags` |
| `aws_cloudwatch_event_permission` | `create_permissions`, on the bus the chart makes | statements of the bus's `policy`, as `PutPermission` writes them: `Sid`, `Principal` (`*` or the account's root), `Action` (`events:PutEvents` by default), `Resource` (the bus's ARN), the `aws:PrincipalOrgID` condition (`condition_org`) |
| `aws_cloudwatch_event_rule` | `create_rules` | `eventbridge/Rule` `rule-<key>`: `name` (`<key>-rule`, `_` as `-`), `eventBusRef` (or `eventBusName`, `create_bus: false`), `description`, `eventPattern`, `scheduleExpression`, `roleRef` (`role_arn: true`), `state` (`enabled` or `state`), `tags` (`tags`, `Name`) |
| `aws_cloudwatch_event_target` | `create_targets` | the rule's `targets`: `id` (`target_id`, else the target's `name`), `arn`, `roleARN` (`attach_role_arn`), `input`, `inputPath`, `inputTransformer`, `runCommandParameters`, `ecsParameters`, `batchParameters`, `kinesisParameters`, `sqsParameters`, `httpParameters`, `deadLetterConfig`, `retryPolicy` |
| `aws_cloudwatch_event_archive` | `create_archives` | `eventbridge/Archive` `archive-<key>`: `name`, `eventSourceRef` (the bus; or `eventSourceARN`), `description`, `eventPattern`, `retentionDays` |
| `aws_cloudwatch_log_delivery_source`, `_destination`, `aws_cloudwatch_log_delivery` | `create_log_delivery` (the source also `create_log_delivery_source`) | `cloudwatchlogs/DeliverySource` `log-source`, `DeliveryDestination` `log-destination-<key>`, `Delivery` `log-delivery-<key>` (by reference to both) |
| `aws_iam_role.eventbridge` | `create_role` | `iam/Role` `role`: named `role_name` (or `bus_name`), trusting `events.amazonaws.com`, `trusted_entities` and, with schedules, `scheduler.amazonaws.com`; `description`, `path`, `permissionsBoundary`, `tags` (`Name`, `tags`, `role_tags`) |
| `aws_iam_policy`, `aws_iam_policy_attachment` (kinesis, kinesis_firehose, sqs, sns, ecs, lambda, sfn, api_destination, cloudwatch, `policy_json`, `policy_jsons`, `policy_statements`) | the `attach_*` values | `iam/Policy` `policy-<name>`, named as the module names them, in the role's `policyRefs` |
| `aws_iam_role_policy_attachment` (`policy`, `policies`); the tracing policy | `attach_policy`, `attach_policies`, `attach_tracing_policy` | the role's `policies` (ARNs). ACK's `Role` takes ARNs or references, not both: with ARNs given beside policies this chart makes, every policy is attached by ARN, built from `account_id` |
| `aws_pipes_pipe` | `create_pipes` | `pipes/Pipe` `pipe-<key>`: `name` (`<key>-pipe`), `roleRef` (or `roleARN`), `source`, `target`, `description`, `desiredState`, `sourceParameters`, `targetParameters`, `enrichment`, `enrichmentParameters`, `tags` |
| `aws_iam_role.eventbridge_pipe`, `aws_iam_policy.service` | a pipe, `create_role` or `create_pipe_role_only`, the pipe's `create_role` not false | `iam/Role` `role-pipe-<key>` (named `role_name_prefix` or the key, trusting `pipes.amazonaws.com` from this account's pipe) and `iam/Policy` `policy-pipe-<key>` (the service integrations `iam_pipes.tf` matches against the pipe's source, enrichment, target and dead-letter queues) |
| `aws_cloudwatch_event_connection` | `create_connections`, `allow_alpha` | ack-extensions `Connection` `connection-<key>` |
| `aws_cloudwatch_event_api_destination` | `create_api_destinations`, `allow_alpha` | ack-extensions `APIDestination` `api-destination-<key>`, `connectionRef` to the connection (`connection_name`, or its own key) |
| `aws_scheduler_schedule_group` | `create_schedule_groups`, `allow_alpha` | ack-extensions `ScheduleGroup` `schedulegroup-<key>`, named `name` or its key (the module reads `name`, so `append_schedule_group_postfix` changes nothing, as in the module), tagged with its own `tags` only |
| `aws_scheduler_schedule` | `create_schedules`, `allow_alpha` | ack-extensions `Schedule` `schedule-<key>`: `groupRef` (a group the chart makes) or `groupName`, `flexibleTimeWindow`, `state`, its `target` (`roleARN`: `role_arn`, else the module's role's ARN, built from `account_id`) |
| `aws_schemas_discoverer` | `create_schemas_discoverer`, `allow_alpha` | ack-extensions Cloud Control `Discoverer` `schemas-discoverer` |

**ARNs the chart builds.** Where ACK takes an ARN and no reference, the
chart writes it, and so needs `account_id` (which Bundt fills from the
Environment) and `region`: the bus's ARN in its policy, log delivery source
and schemas discoverer (and an existing bus's for an archive); the module
role's ARN in a target (`attach_role_arn: true`) or schedule; a pipe role's
trust (`aws:SourceAccount`, `aws:SourceArn`); the API destination policy
(`api-destination/<name>/*`: the ARN ends in an ID AWS gives); and the
chart's own policies attached beside policy ARNs. The log delivery source
is made by default (`create_log_delivery`, `create_log_delivery_source`),
so a default render needs both.

**The chart owns whole objects.** A rule's targets are inline in ACK's
`Rule`, and the permissions are the bus's whole `policy`: a target or a
statement added outside the component is removed.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`bus`, `role`), `namePrefix`, `labels` (Kubernetes
only), `allow_alpha`, `account_id` and `partition` (which the module reads
from the provider) and `deletion_policy`; `tags` is the module's own, and
`region` becomes the annotation `services.k8s.aws/region`. `bus_name:
default` with `create_bus: true` (the module's defaults) stops the render:
every account has a default bus, which cannot be created; set
`create_bus: false` to use it.

Refused or accepted in part (`upstream.yaml`):

- `bus_description`, `dead_letter_config`, `kms_key_identifier`,
  `log_config`: ACK's `EventBus` (eventbridge-controller v1.9.2) has only
  `name`, `eventSourceName`, `policy` and `tags`. Without `log_config` the
  log delivery source takes `ERROR_LOGS`, as the module does when it is
  null.
- `schedule_group_timeouts`: Terraform's (by design).
- `rules`, in part: a rule's `name_prefix` (a random suffix) and
  `force_destroy` (ACK's `Rule` has no `ForceDelete`).
- `targets`, in part: `destination` (an API destination this chart makes:
  its ARN ends in an ID AWS gives, and a rule's targets take no reference;
  give the ARN as `arn`), `appsync_target` (no `AppSyncParameters` in ACK's
  targets) and `force_destroy`. `target_id` defaults to the target's name
  (Terraform: a random ID).
- `archives`, in part: `kms_key_identifier` (no field in ACK's `Archive`).
- `permissions`, in part: only on the bus the chart makes. Neither ACK nor
  ack-extensions has a `Permission` kind, so a permission on the default or
  another existing bus is refused.
- `connections`, in part: the secret values (`api_key.value`,
  `basic.password`, `oauth.client_parameters.client_secret`) are
  references to Kubernetes Secrets, `{secretName, key}`
  ([`docs/secrets.md`](../../../docs/secrets.md)); a string is refused.
  `kms_key_identifier` is refused (no field in ack-extensions'
  `Connection`).
- `create_connections`, `create_api_destinations`,
  `create_schemas_discoverer`, `schedule_groups`, `schedules`: ack-extensions
  kinds, behind `allow_alpha: true`. A group's or schedule's `name_prefix`
  is refused.
- `pipes`, in part: `kms_key_identifier` and `log_configuration` (ACK's
  `Pipe`, pipes-controller v1.8.0, has neither), and an `enrichment` naming
  an API destination this chart makes (no reference; give its ARN).
  `role_name_prefix` names the role as given, without the random suffix.
  A self-managed Kafka source's subnets are read from `vpc.vpc`, as the
  module reads them, or `vpc.subnets`.
- `attach_tracing_policy`, in part: the role gets AWS's managed
  `AWSXrayWriteOnlyAccess` by ARN, where the module copies its document
  into a policy of its own.
- `role_force_detach_policies`, in part: `true` only; ACK always detaches a
  role's managed policies before deleting it.

## Outputs

All 34 of the module's outputs but `eventbridge_schedule_group_states`
(ack-extensions' `ScheduleGroup` status has no state): the bus's name and
ARN; the archives', connections', API destinations', rules', schedule
groups', schedules' and pipes' IDs and ARNs by key (a rule's ID is
`<bus>/<rule>`, or the rule's name on the default bus, as the provider
gives it; a permission's `<bus>/<statement_id>`); the roles' names and
ARNs; the log delivery source's name and ARN; and the whole-resource
outputs (`eventbridge_bus`, `_rules`, `_targets`, ...) as the objects'
specs. `""` or `{}` for what the component did not make, as the module
gives.

## Tests

`task test CHART=aws/eventbridge` renders every `tests/*.values.yaml`
against its golden file and the pinned controllers' and ack-extensions'
CRDs, and checks each `tests/*.fail.yaml` is refused with its message.
`tests/mock.values.yaml` is the fake-AWS case (`task test-mock-aws`,
`hack/mock-apply-aws.sh`): a bus with a permission, a rule to an SQS
queue, an archive, a log delivery to a log group, the role with its SQS
policy and a pipe between two queues with its role.
