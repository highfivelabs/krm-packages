# estate-read

The read-only IAM role Bundt's estate collector assumes in one AWS account,
`bundt-estate-read`, as AWS Controllers for Kubernetes (ACK) objects: AWS's
managed `job-function/ViewOnlyAccess` and a policy of its own with Cloud
Control's read actions, trusted by the collector's principal only. Bundt
Package `aws-estate-read` (`package.yaml`). Checked against ACK's
iam-controller v1.11.3 (`UPSTREAM.md`, "ACK controllers").

It is not a module port, so it has no `upstream.yaml`: `Chart.yaml` marks
it `hfl.dev/kind: wrapper`. It depends on the catalog's
[`iam-role`](../iam-role/README.md) chart for its policy document builder
(`iamrole.document`) and renders the `Role` and the `Policy` as `iam-role`
and [`iam-policy`](../iam-policy/README.md) render them. The `iam-role`
dependency renders nothing of its own (`iam-role.create: false`; the render
stops if it is set).

## Why a chart

Bundt's estate inventory reads what already runs in a customer's clouds
(bundt `drafts/estate/README.md`). On AWS one of its three sources is a
read-only role in every account, assumed in turn, listing with Cloud
Control `ListResources` (section 3). The draft printed an AWS
CloudFormation StackSet for that role; the builder's rule (bundt
`asks/estate-answers.md`, "The bootstrap roles", 2026-11-13) is that such
setup is delivered as Bundt infra charts and stacks from the catalog, on
ACK, Config Connector and ASO, deployed through Bundt like anything else,
never as CLI commands or CloudFormation. This chart is that role: a Bundt
Instance of it per account (or one per account in a Stack), in place of
the StackSet.

## Example

```yaml
account_id: "111122223333"
collector:
  aws_principal_arns: ["arn:aws:iam::444455556666:role/bundt-estate-collector"]
```

renders the role `bundt-estate-read`, which only the collector's role may
assume (`sts:AssumeRole`), with `ViewOnlyAccess` and the policy
`bundt-estate-read-cloud-control` attached (`tests/golden/basic.yaml`).

## Values

| Value | Renders |
| --- | --- |
| `name` (or `names.role`) | the role's name, default `bundt-estate-read`: the same in every account, so the collector finds it; `namePrefix` goes in front unless `names.role` is set |
| `path`, `description`, `max_session_duration` | the role's (and policy's) path, default `/`; its description; its session limit, default 3600 seconds (3600 to 43200) |
| `collector.aws_principal_arns` | IAM principal ARNs (the collector's role) allowed `sts:AssumeRole` (statement `Collector`) |
| `collector.external_id` | optional: an `sts:ExternalId` the AWS principals must present |
| `collector.google_service_account_unique_id`, `collector.google_audience` | the collector on GKE: its Google service account's unique ID (21 digits, quoted) and the audience of the ID tokens it presents, allowed `sts:AssumeRoleWithWebIdentity` from IAM's built-in `accounts.google.com` principal, conditioned on `accounts.google.com:sub` and `accounts.google.com:oaud` (statement `CollectorGoogle`; the same trust as [`google-federated-role`](../google-federated-role/README.md), no IAM OIDC provider) |
| `view_only_access` | attach `arn:<partition>:iam::aws:policy/job-function/ViewOnlyAccess`, default `true` |
| `cloud_control_read` | make and attach `<name>-cloud-control`, allowing `cloudformation:ListResources` and `cloudformation:GetResource` on `*`, default `true` |
| `account_id`, `partition` | with both policies, the Cloud Control policy's ARN is built from them (ACK's `Role` takes policy ARNs or references, not both); Bundt fills `account_id` from the Environment |
| `tags`, `labels`, `namePrefix`, `names`, `deletion_policy` | as every catalog chart (docs/design.md, "Names, labels and tags") |

At least one trust is required; the render stops when the role would trust
no one, or read nothing. The trust never names a Bundt provisioning
(`bnt-ip`) identity: the estate identity and the provisioning identities
are different principals (the draft's section 6).

**Not `ReadOnlyAccess`**: it reads data too (object contents, table items).
`ViewOnlyAccess` lists and describes. Whether it also covers Cloud Control's
list handlers for the common types is the draft's assumption E7, not yet
checked; the Cloud Control policy adds the two Cloud Control API actions
themselves (assumed, see below), and anything a handler needs beyond
`ViewOnlyAccess` is found when the collector first runs.

## Assumptions

- **ER1.** (likely, cheap) The per-account source needs Cloud Control's
  own `cloudformation:ListResources` and `cloudformation:GetResource`
  besides the handlers' permissions; the chart grants both explicitly
  rather than rely on `ViewOnlyAccess`.
- **ER2.** (likely, cheap) One hour is enough for one collector run per
  account (`max_session_duration`); a run assumes the role again.

## Outputs

`name`, `arn` (what the collector assumes), `unique_id` (IAM's RoleId).

## Every account of an organization

A StackSet deploys one stack to every account of an organization or OU,
and to accounts that join later. Bundt's equivalent is one Instance of
this chart per account, each in that account's Environment (ACK's
cross-account annotations, a Bundt provider concern; `docs/aws.md`,
section 8). Following an organization's membership (new accounts) is a
gap for Bundt and ack-extensions, not a reason to fall back to a StackSet
(bundt `asks/estate-answers.md`).

## Tests

`task test CHART=aws/estate-read` builds the dependency from `../iam-role`
(`Chart.lock`; nothing is fetched) and renders seven cases (`basic`,
`google`, `full`, `view-only`, `cloud-control`, `names`, `mock`), each
validated against iam-controller v1.11.3's CRDs and with a namePrefix, and
the refusals in `tests/*.fail.yaml`. `tests/mock.values.yaml` is the case
the fake-AWS run applies (`task test-mock-aws MOCK_CHARTS=estate-read`).
Not deployed: no real account holds it yet.

## Changes

- **0.1.0** (2026-11-15): the role, its two trusts, `ViewOnlyAccess` and
  the Cloud Control read policy.
