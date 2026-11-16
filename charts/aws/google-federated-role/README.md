# google-federated-role

An IAM role that one Google service account assumes through IAM's built-in
Google federation, with its managed policies, as AWS Controllers for
Kubernetes (ACK) objects. Bundt Package `aws-google-federated-role`
(`package.yaml`). Checked against ACK's iam-controller v1.11.3
(`UPSTREAM.md`, "ACK controllers").

It is not a module port, so it has no `upstream.yaml`: `Chart.yaml` marks
it `hfl.dev/kind: wrapper`. It depends on the catalog's
[`iam-role`](../iam-role/README.md) chart for its policy document builder
(`iamrole.document`, which builds a document as terraform-aws-modules'
`aws_iam_policy_document` does), and renders the `Role` and each `Policy`
as `iam-role` and [`iam-policy`](../iam-policy/README.md) render them. The
`iam-role` dependency renders nothing of its own (`iam-role.create: false`;
the render stops if it is set).

This is how setup that needs a Google workload to act in AWS is delivered:
through Bundt and the catalog, never a hand-written CloudFormation stack or
CLI commands (bundt `asks/estate-answers.md`, "The bootstrap roles").

## Example

```yaml
name: gsa-reader
google_service_account_unique_id: "123456789012345678901"
audience: gsa-reader
policies:
  read:
    description: Read one bucket
    statements:
      - sid: Read
        actions: ["s3:GetObject", "s3:ListBucket"]
        resources: ["arn:aws:s3:::example", "arn:aws:s3:::example/*"]
```

renders the role `gsa-reader`, trusting only that service account's ID
tokens with the audience `gsa-reader`, and the policy `gsa-reader-read`
attached to it by reference (`tests/golden/basic.yaml`).

## The trust

```json
{"Sid": "GoogleServiceAccount", "Effect": "Allow",
 "Principal": {"Federated": ["accounts.google.com"]},
 "Action": ["sts:AssumeRoleWithWebIdentity"],
 "Condition": {"StringEquals": {
   "accounts.google.com:oaud": ["<audience>"],
   "accounts.google.com:sub": ["<unique ID>"]}}}
```

IAM names a Google ID token's claims this way for its built-in
`accounts.google.com` principal: `accounts.google.com:sub` is the token's
`sub` (the service account's unique ID, which no other principal's token
carries), `accounts.google.com:oaud` its `aud` (the audience the workload
asked Google's metadata server for), and `accounts.google.com:aud` its
`azp`. The chart makes **no IAM OIDC provider**: one for
`https://accounts.google.com` takes over every Google ID token in the
account and refuses any whose audience is not on its client ID list, which
broke another Google-federated role in the AWS test account on 2026-11-15
(ack-extensions `docs/live-runs.md`).

## Values

| Value | Renders |
| --- | --- |
| `name` (or `names.role`) | the role's name (required); `namePrefix` goes in front unless `names.role` is set |
| `path`, `description`, `max_session_duration` | the role's (and its policies') path, default `/`; its description; its session limit, 3600 to 43200 seconds |
| `google_service_account_unique_id` | the service account's unique ID, 21 digits, quoted (`gcloud iam service-accounts describe EMAIL --format='value(uniqueId)'`); required |
| `audience` | the token's audience the trust checks; required |
| `trust_sid` | the trust statement's `Sid`, default `GoogleServiceAccount` |
| `policies` | `{key: {description, statements}}`: one `Policy` per key, named `<role's name>-<key>`, its document built from `statements` in the iam module's shape (as `iam-role`'s `inline_policy_permissions`); each document at most IAM's 6,144 characters, or the render stops |
| `policy_arns` | `{key: ARN}`: AWS managed or existing policies attached, in key order |
| `account_id`, `partition` | needed only when `policies` and `policy_arns` are both set: ACK's `Role` takes policy ARNs or references, not both, so the policies' ARNs are then built (`arn:<partition>:iam::<account_id>:policy<path><name>`); otherwise the policies are attached by reference (`policyRefs`), which also orders them before the role |
| `tags`, `labels`, `namePrefix`, `names`, `deletion_policy` | as every catalog chart (docs/design.md, "Names, labels and tags"); `tags` go on the role and every policy |

At most 10 policies in all (IAM's default quota of managed policies per
role); the render stops beyond it.

## Outputs

`name`, `arn`, `unique_id` (IAM's RoleId), `policy_arns` (the ARNs of the
policies it made, sorted).

## The ack-extensions live-test role

**Status: the ack-extensions-live-tests instance is deployed and adopted
(2026-11-16).** ACK's iam-controller adopted the role the CloudFormation
stack had made, in place (same name and RoleId; ack-extensions
`docs/live-runs.md`, AWSROLE-ADOPT), and the stack let go of it; the role
is this instance's now. Applied again the same day with the ACK-kind
suite's statements (HFL-623): ACK adopted the role and `part-1` again,
updated `part-1`'s document and made and attached `part-2`; all three
objects `ACK.ResourceSynced`. A `Policy` is adopted by its ARN, not its
name: each carries `services.k8s.aws/adoption-fields: {"arn": ...}` beside
`adopt-or-create` (without it ACK called `CreatePolicy` and was refused
`EntityAlreadyExists`).

`tests/ack-extensions-live-tests.values.yaml` is the instance for
ack-extensions' live tests: the role `ack-extensions-live-tests` in the AWS
test account, which the offload VMs' Google service account assumes for
`task test-live` (ack-extensions `docs/live-runs.md`). When it was adopted
it carried the same role, trust and permissions as ack-extensions'
CloudFormation template at `7d67acc` (2026-11-15), compared statement for
statement (trust policy, name, description, session limit, tag and the one
managed policy's document and description all equal), with two
differences:

- the managed policy is named `ack-extensions-live-tests-part-1`, where
  CloudFormation generated a name;
- the template's monthly budget alarm (`AWS::Budgets::Budget`) is not
  here: no ACK controller or ack-extensions kind serves AWS Budgets yet,
  a gap for ack-extensions. The alarm stays in that stack.

Its `policies` are ack-extensions' generated
`cloudcontrol/config/live/role-policies.yaml`, statement for statement:
the Cloud Control kinds sampled on live runs (`liveKinds`), the sweeper,
the tests' fixtures, and (since 2026-11-16, HFL-623) the ACK-kind suite's
calls, which take a second policy, `part-2`. When those change there, the
statements are copied here and the instance is applied again through ACK
the way it was adopted (`adoption-policy: adopt-or-create` on every object,
and `adoption-fields` with the ARN on each existing `Policy`, so the role
and its policies are adopted again; `deletion_policy: ABANDON`; ACK's
resource tags off). A check that the two files agree is not built yet.

The unique ID in the file is a placeholder (`000000000000000000000`): this
repository is public, and identifiers come from the environment. Set the
real one where the instance is deployed.

## Tests

`task test CHART=aws/google-federated-role` builds the dependency from
`../iam-role` (`Chart.lock`; nothing is fetched) and renders six cases
(`basic`, `full`, `arns`, `names`, `mock`, `ack-extensions-live-tests`),
each validated against iam-controller v1.11.3's CRDs and with a
namePrefix, and the refusals in `tests/*.fail.yaml`.
`tests/mock.values.yaml` is the case the fake-AWS run applies
(`task test-mock-aws MOCK_CHARTS=google-federated-role`). The trust itself
(a Google token exchanged for the role's credentials) can only be shown on
real AWS.

## Changes

- **0.1.0** (2026-11-15): the role, its Google trust, its policies by
  reference or ARN, and the ack-extensions live-test instance.
