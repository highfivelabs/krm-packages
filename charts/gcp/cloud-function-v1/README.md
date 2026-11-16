# cloud-function-v1

A Cloud Function (1st gen) built from a bundle in Cloud Storage, with its
optional source bucket, service account, VPC Access connector and IAM, as
Config Connector objects. A port of Cloud Foundation Fabric's
[`modules/cloud-function-v1`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/cloud-function-v1)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-cloud-function-v1` (`package.yaml`). Checked against Config Connector
v1.160.0.

## Example

```yaml
project_id: my-project
region: europe-west1
name: test-cf-http
bucket_name: my-bucket
bundle_config: {path: "gs://my-bucket/bundles/function.zip"}
```

renders the `CloudFunctionsFunction` `test-cf-http` (an HTTPS trigger),
its `IAMServiceAccount` and the account's two project roles
(`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names.function`, `names.bucket`, `names.serviceAccount`, `names.vpcConnector` | not fabric variables: those names in full, used as given (no `namePrefix`, no `prefix`) and checked (63, 63, 30 and 25 characters; a leading letter but for the bucket) (`docs/design.md`, "Names, labels and tags") |
| `namePrefix` | not a fabric variable: an optional convenience; when set, every derived name starts with `<namePrefix>-`. `bucket_name` without `bucket_config` names a bucket the chart does not make and is never changed. Empty changes nothing |
| `labels` | Kubernetes labels on every object; an unprefixed key also becomes a Google Cloud label (fabric's `labels`, on the function and the bucket, are these) |
| `tags` | cloud labels on every object (Google Cloud labels) |
| `deletion_policy` | not a fabric variable: `DELETE` (default) or `ABANDON`; `PREVENT` is refused (`docs/design.md`, "Deletion") |
| `project_id`, `region`, `name`, `prefix`, `description` | the function's project, region, name (`<prefix>-<name>`) and description |
| `bundle_config` | `path`, a Cloud Storage object URI `gs://<bucket>/<object>.zip`, is the function's `sourceArchiveUrl`: the object, in the bucket fabric's `local.bucket` names (below). A local zip or folder, and `folder_options`, are **refused** (below) |
| `bucket_name`, `bucket_config` | the source bucket. With `bucket_config`, a `StorageBucket` `<prefix>-<bucket_name>` (uniform access, in `bucket_config.location` or the region; `lifecycle_delete_age_days` adds a Delete rule for archived versions and turns versioning on; `force_destroy` is Config Connector's `force-destroy` annotation). Without, `bucket_name` as given |
| `function_config` | `runtime`, `entry_point`, `memory_mb` (`availableMemoryMb`), `instance_count` (`maxInstances`), `timeout_seconds` (`timeout`, `<n>s`). `cpu` is accepted and unused, as in fabric's module (a 1st gen function's CPU follows its memory) |
| `environment_variables`, `ingress_settings` | `environmentVariables`, `ingressSettings` |
| `trigger_config`, `https_security_level` | without `trigger_config`, an `httpsTrigger` with `securityLevel` (`SECURE_ALWAYS` when null), as fabric's `trigger_http`; with it, an `eventTrigger`: `event` (`eventType`), `resource` (`resourceRef.external`), `retry` (`failurePolicy`) |
| `vpc_connector`, `vpc_connector_create` | `vpcConnectorRef` (the created connector by reference, else `vpc_connector.name`) and `vpcConnectorEgressSettings`; `vpc_connector_create` a `VPCAccessConnector` named its `name` or `name` |
| `service_account_config` | with `create` (the default), an `IAMServiceAccount` (`name` or `name`; `display_name`, else the account's name), the function's `serviceAccountRef`, and one `IAMPolicyMember` on the project per `roles` entry; else `email`, if any, runs the function |
| `iam` | one `IAMBinding` of config-connector-extensions (`iam.gcp.hfl.dev/v1beta1`) per role on the function, by its full name (`projects/<project>/locations/<region>/functions/<name>`): exact, as fabric's `google_cloudfunctions_function_iam_binding` (`docs/design.md`, "IAM") |
| `repository_settings` | only fabric's default, `registry: ARTIFACT_REGISTRY` and no `repository`, which is Google Cloud's default too; another registry or a `repository` is **refused** |
| `build_environment_variables`, `build_worker_pool`, `kms_key`, `secrets` | **refused** when set: Config Connector's `CloudFunctionsFunction` (v1.160.0) has no field for them, and config-connector-extensions has no 1st gen function kind. The `cloud-function-v2` chart takes all four |
| `context` | **refused**: Bundt's `${...}` expressions and `valueFrom` replace fabric's `$context` lookups |

## Outputs

| Output | From |
| --- | --- |
| `bucket_name` | the bucket of the function's source (fabric's `local.bucket`) |
| `function_name` | the function's name |
| `id` | `projects/<project>/locations/<region>/functions/<name>` |
| `invoke_command` | for an HTTP function (no `trigger_config`), fabric's `curl` command with the function's URL, `status.httpsTrigger.url`; null until the function reports it |
| `service_account_email`, `service_account_iam_email` | the created account's email from its status, else `service_account_config.email`; the second prefixed `serviceAccount:` |
| `vpc_connector` | `projects/<project>/locations/<region>/connectors/<name>` of the created connector, else null |

These are fabric's declared outputs, selected (`docs/design.md`,
"Outputs"); `bucket`, `function` and `service_account` (whole Terraform
objects) are not outputs.

## Differences from the module

- **The bundle must already be in Cloud Storage.** fabric zips a local
  folder (`archive_file`) or uploads a local zip
  (`google_storage_bucket_object`) from the machine running Terraform. A
  chart has no files to read, and config-connector-extensions'
  `StorageBucketObject` (`v1alpha1`) takes text content, not an archive, so
  a path that does not start with `gs://` and `folder_options` are refused
  by name. Upload the bundle (a CI step, `gcloud storage cp`) and give its
  URI; the function's source is that object in the bucket fabric's
  `local.bucket` names, as fabric's is. (The `cloud-function-v2` chart
  makes the same choice.)
- **Build settings, CMEK and secrets are refused.** Config Connector's
  `CloudFunctionsFunction` has no `buildEnvironmentVariables`,
  `buildWorkerPool`, `kmsKeyName`, `dockerRegistry`, `dockerRepository`,
  `secretEnvironmentVariables` or `secretVolumes`. A function that needs
  them is a 2nd gen function (`cloud-function-v2`).
- fabric's `replace_triggered_by` on the IAM binding has no equivalent:
  the `IAMBinding` names the function by its full name, and a new function
  of the same name gets the same binding.
- The service account and a created VPC connector are referenced by
  object (`serviceAccountRef.name`, `vpcConnectorRef.name`), so the
  function waits for them.

## From Terraform

A `module "x" { source = ".../modules/cloud-function-v1" ... }` block's
arguments become the values unchanged, except refused fields: a local
`bundle_config.path` must become the `gs://` URI of the uploaded bundle
(fabric's own upload names it `bundle-<md5>.zip`, or the zip's basename, in
the bucket). To adopt an existing function, bucket and account, keep their
names: Config Connector acquires the resources of those names
(`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/cloud-function-v1`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/cloud-function-v1)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml) records
the repository, path, release, commit, licence and the date last synced, and
the module is listed in [`upstream/modules.yaml`](../../../upstream/modules.yaml)
with every other module the catalog tracks. To take a newer release, see
[`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=cloud-function-v1`: render cases `basic`,
`pubsub-trigger`, `iam`, `bucket` and `service-account` (fabric's README
examples, with the bundle in Cloud Storage), `full` (every field the chart
carries), `names` and `mock`, each validated against Config Connector
v1.160.0's CRD schemas, and the refusals in `tests/*.fail.yaml`.

Applied to the fake cloud on 2026-11-15 on an offload VM (`task test-mock
MOCK_CHARTS="cloud-function-v1 workstation-cluster"`, run
`off-catalog-1115111105-4763`, at config-connector-extensions `384d07c0`):
`tests/mock.values.yaml`, an HTTP function, the bucket the chart makes for
its source and its service account with one project role: all 4 objects
Ready, the outputs evaluated, all deleted. mockgcp reports an empty
function URL, so `invoke_command` is null there (the run showed a command
with no URL; the output now waits for a non-empty one). The VPC Access
connector is render-tested only.

Not applied live.

## Changes

- **0.2.0** (2026-11-15): `iam` renders config-connector-extensions'
  `IAMBinding` per role on the function (exact, as fabric's binding),
  instead of an `IAMPartialPolicy`, which left members added out of band;
  config-connector-extensions 81c3f9b5 (its #138) takes a
  `CloudFunctionsFunction`.
- **0.1.0** (2026-11-15): the port: the function from a Cloud Storage
  bundle, its source bucket, service account, VPC Access connector, and
  IAM on the function.
