# ecr

An ECR repository with its repository and lifecycle policies, and the
registry's pull through cache rules, policy, scanning and replication
configurations, as AWS Controllers for Kubernetes (ACK) objects. A port of
terraform-aws-modules'
[`terraform-aws-ecr`](https://github.com/terraform-aws-modules/terraform-aws-ecr/tree/b6ef04d088cf91d5ba9505132e9ff7c9f847ed5d)
at v3.2.0 (`b6ef04d0`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-ecr`
(`package.yaml`). Checked against ACK's ecr-controller v1.11.1
(`UPSTREAM.md`, "ACK controllers") and ack-extensions' `RegistryPolicy`,
`RegistryScanningConfiguration` (Cloud Control) and
`ReplicationConfiguration` (`crds/ack-extensions`).

## Example

The module's README usage ("Private Repository"):

```yaml
repository_name: private-example
repository_read_write_access_arns: ["arn:aws:iam::012345678901:role/terraform"]
repository_lifecycle_policy: '{"rules":[{"rulePriority":1,"description":"Keep last 30 images","selection":{"tagStatus":"tagged","tagPrefixList":["v"],"countType":"imageCountMoreThan","countNumber":30},"action":{"type":"expire"}}]}'
tags: {Terraform: "true", Environment: dev}
```

renders one `ecr.services.k8s.aws/v1alpha1` `Repository` (`hfl.dev/id`
`repository`) with its policy and lifecycle policy inline
(`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_ecr_repository.this` | `Repository` `repository`: `name`, `encryptionConfiguration` (with `repository_encryption_type` or `repository_kms_key`), `imageScanningConfiguration`, `imageTagMutability` and its exclusion filters, `tags` |
| `aws_ecr_repository_policy.this` | the `Repository`'s `policy` (`attach_repository_policy`) |
| `aws_ecr_lifecycle_policy.this` | the `Repository`'s `lifecyclePolicy` (`create_lifecycle_policy` with a non-empty `repository_lifecycle_policy`) |
| `aws_ecr_pull_through_cache_rule.this` | one `PullThroughCacheRule` per entry of `registry_pull_through_cache_rules` (`hfl.dev/id` `pull_through_cache_rule:<key>`), in the entry's `region` or the chart's |
| `aws_ecr_registry_policy.this` | ack-extensions' `RegistryPolicy` (Cloud Control), with `allow_alpha: true` |
| `aws_ecr_registry_scanning_configuration.this` | ack-extensions' `RegistryScanningConfiguration` (Cloud Control), with `allow_alpha: true` |
| `aws_ecr_replication_configuration.this` | ack-extensions' `ReplicationConfiguration` (`ecr.aws.hfl.dev`), with `allow_alpha: true` |
| `aws_ecrpublic_repository.this`, `aws_ecrpublic_repository_policy.example` | refused (below) |

ACK's `Repository` takes both policies inline, where Terraform has
standalone resources: the chart owns each whole field, and a second
Package cannot add to a repository's policy (`docs/aws.md`, "Porting
notes"). The registry policy, scanning and replication configurations are
one per account and Region (replication: per account): two components
that set one overwrite each other, as two Terraform stacks would.

The repository policy is built as the module's `aws_iam_policy_document`
builds it: `PrivateReadOnly` (for `repository_read_access_arns` and
`repository_read_write_access_arns`, or the account's root when both are
empty), `PrivateLambdaReadOnly` (for `repository_lambda_read_access_arns`),
`ReadWrite` (for `repository_read_write_access_arns`), then
`repository_policy_statements` in key order, each with the `sid` it gives
(the module gives none by default). With `create_repository_policy: false`
the chart attaches `repository_policy` as given.

`region` becomes the annotation `services.k8s.aws/region`; unset, ACK
takes the namespace's or its controller's region.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`repository`), `namePrefix` (on the repository's name
and each pull through cache rule's repository prefix), `labels`
(Kubernetes only), `deletion_policy` (`ABANDON` renders
`services.k8s.aws/deletion-policy: retain`), `allow_alpha`, and
`account_id` and `partition`; `tags` is the module's own.

- `repository_name` (or `names.repository`) is required.
- `account_id` is required for the default policy without read or
  read/write ARNs: its principal is `arn:<partition>:iam::<account_id>:root`,
  which the module reads from the provider. Bundt fills `account_id` from
  the Environment.
- `repository_lifecycle_policy` empty (the default) renders no lifecycle
  policy, where Terraform would fail on an empty document.

Refused (`upstream.yaml`):

- `repository_type: public` and `public_repository_catalog_data`: ACK's
  ecrpublic `Repository` (ecrpublic-controller v1.7.0) has no policy or
  catalog data field, and the module always attaches a policy to a public
  repository.
- `repository_force_delete`: ACK's `Repository` has no field to delete a
  repository that still holds images.

## Outputs

`repository_name`, `repository_arn`, `repository_registry_id` (a private
repository's registry is its account) and `repository_url` (ACK's
`repositoryURI`).

## Tests

`task test CHART=aws/ecr`: golden renders validated against ecr-controller
v1.11.1's CRDs and ack-extensions', and the refusal cases.
`tests/mock.values.yaml` is the case the fake-AWS run applies: the
repository with the default policy for the fake's own account
(`tests/mock.set.yaml`), a lifecycle policy and a pull
through cache rule. The registry settings are render-tested only: they are
one per account, and the Cloud Control kinds have not yet become Ready on
either fake (`docs/aws.md`, "Charts").
