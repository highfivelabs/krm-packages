# secrets-manager

A Secrets Manager secret with its replicas, its value and its resource
policy, as AWS Controllers for Kubernetes (ACK) objects. A port of
terraform-aws-modules'
[`terraform-aws-secrets-manager`](https://github.com/terraform-aws-modules/terraform-aws-secrets-manager/tree/f3d3da1c801816563080065c5f37f212c8dee444)
at v2.2.0 (`f3d3da1c`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-secrets-manager`
(`package.yaml`). Checked against ACK's secretsmanager-controller v1.8.2
(`UPSTREAM.md`, "ACK controllers") and ack-extensions' Cloud Control
`ResourcePolicy` (`crds/ack-extensions`).

## Example

The module's README usage, with the value from a Kubernetes Secret the
team provides in place of `secret_string`:

```yaml
name: example
description: Example Secrets Manager secret
recovery_window_in_days: 0
replica:
  us-west-2: {}
secret_string_ref: {name: example-secret, key: value}
tags: {Environment: Development, Project: Example}
```

renders one `secretsmanager.services.k8s.aws/v1alpha1` `Secret`
(`hfl.dev/id` `secret`) (`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_secretsmanager_secret.this` | `Secret` `secret`: `name`, `description`, `kmsKeyID`, `forceOverwriteReplicaSecret`, `recoveryWindowInDays`, `replicaRegions` (one per `replica` entry, its region the entry's `region` or key), `tags` |
| `aws_secretsmanager_secret_version.this` | the `Secret`'s `secretString`: a key of a Kubernetes Secret (`secret_string_ref`), when `create_secret_version` is true |
| `aws_secretsmanager_secret_policy.this` | ack-extensions' `ResourcePolicy` `policy` (`secretsmanager.cloudcontrol.aws.hfl.dev`, Cloud Control), with `allow_alpha: true` |
| `aws_secretsmanager_secret_rotation.this` | refused (below) |

**A secret's value never enters the render.** The module takes the value
as a variable (`secret_string`, `secret_string_wo`, `secret_binary`) or
generates one (`create_random_password`); a chart would write any of them
into the rendered objects, which Bundt stores and shows. ACK's `Secret`
takes its value from a Kubernetes Secret instead, so the chart adds
`secret_string_ref: {name, key, namespace}`, a key of a Kubernetes Secret
(in the component's namespace unless `namespace` is set) that the team
provides, by hand, from another secret store or from a generator. Without
it the secret has no value until something else writes one, as with the
module's `create_secret_version: false`.

The resource policy is built as the module's `aws_iam_policy_document`
builds it: the source documents, then the statements (in key order, each
`sid` defaulting to its key), then the override documents, merged by
`Sid`. Like the module, the chart gives a statement no default resource.
The `ResourcePolicy` names the secret by name (`secretId`; the kind has no
reference field), and Cloud Control retries it until the secret exists.

`region` becomes the annotation `services.k8s.aws/region`; unset, ACK
takes the namespace's or its controller's region.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`secret`), `namePrefix`, `labels` (Kubernetes only),
`deletion_policy` (`ABANDON` renders `services.k8s.aws/deletion-policy:
retain`), `allow_alpha` and `secret_string_ref`; `tags` is the module's
own.

- `name` (or `names.secret`) is required: Terraform would generate a name
  (`name_prefix`), which would change between renders.

Refused (`upstream.yaml`):

- `secret_string`, `secret_string_wo`, `secret_string_wo_version`,
  `secret_binary`: a value in values would be written into the render
  (and ACK's `Secret` has no binary field); use `secret_string_ref`.
- `create_random_password`, `random_password_length`,
  `random_password_override_special`: a chart cannot make a random value
  without writing it into the render, and ACK's `Secret` cannot generate
  one; generate it into a Kubernetes Secret and set `secret_string_ref`.
- `version_stages`: ACK's `Secret` has no staging labels field.
- `ignore_secret_changes`: ACK writes the Kubernetes Secret's value; it
  cannot leave a value changed outside it alone.
- `enable_rotation`, `rotate_immediately`, `rotation_lambda_arn`,
  `rotation_rules`: rotation has no ACK field (secretsmanager-controller
  v1.8.2) and no ack-extensions kind at the catalog's `ACKX_COMMIT`
  (`docs/aws.md`, section 5, gap 2).
- `name_prefix`: Terraform's random suffix would change between renders.

## Outputs

`secret_arn` and `secret_id` (Terraform's id of a secret is its ARN),
`secret_name`, `secret_replica` (ACK's `replicationStatus`),
`secret_version_id`. Not provided: `secret_string` and `secret_binary`, as
a secret's value never enters a Package's outputs.

## Tests

`task test CHART=aws/secrets-manager`: golden renders validated against
secretsmanager-controller v1.8.2's CRDs and ack-extensions'
`ResourcePolicy`, and the refusal cases. `tests/mock.values.yaml` is the
case the fake-AWS run applies, its value from the Kubernetes Secret in
`tests/mock.needs.yaml` (`docs/aws.md`, "Charts").

The fake-AWS run (`off-catalog-1115093833-20939`, 2026-11-15): the `Secret`
synced with its value from the Kubernetes Secret, its outputs evaluated
and it was deleted, on AWS Emulator and on moto. The `ResourcePolicy` did
not become Ready on either fake, as no Cloud Control kind has yet
(`docs/aws.md`, "Charts"), so the resource policy is proven by render tests
only.
