# certificate-authority-service

A Certificate Authority Service CA pool, its certificate authorities and
IAM on the pool, as Config Connector objects. A port of Cloud Foundation
Fabric's
[`modules/certificate-authority-service`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/certificate-authority-service)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-certificate-authority-service` (`package.yaml`). Checked against
Config Connector v1.160.0's `PrivateCACAPool` and
`PrivateCACertificateAuthority`, and config-connector-extensions'
`IAMBinding` (`iam.gcp.hfl.dev/v1beta1`) and `IAMMember`
(`iam.gcp.hfl.dev/v1beta1`), at `81c3f9b5`: Config Connector's own IAM
kinds cannot reference a CA pool.

## Example

```yaml
project_id: my-project
location: europe-west1
ca_pool_config:
  create_pool:
    name: test-ca
```

renders the `PrivateCACAPool` `test-ca` (DevOps tier) and fabric's default
CA, `test-ca`, self-signed, RSA 2048, for `test.example.com`
(`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.ca_pool` sets the pool's name in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the pool's name (unless set under `names`) starts with `<namePrefix>-`. CAs are named by their keys, inside the pool |
| `labels`, `tags` | not fabric variables: Kubernetes labels on every object; unprefixed ones, and `tags`, are Google Cloud labels on the pool and the CAs |
| `project_id` | every object's project annotation and `projectRef` |
| `location` | every object's `location` |
| `ca_pool_config` | `create_pool`: one `PrivateCACAPool`, tier `ENTERPRISE` with `enterprise_tier`, else `DEVOPS`, with `publishingOptions` (`publish_ca_cert`, `publish_crl`; `encoding_format` in part, below). `use_pool.id`: no pool; the CAs and IAM go on that pool, as fabric decides (it takes precedence) |
| `ca_configs` | one `PrivateCACertificateAuthority` per key, its ID the key, in the pool: type `SELF_SIGNED` when `is_self_signed` and no `subordinate_config`, else `SUBORDINATE`; `lifetime`; `gcs_bucket` as `gcsBucketRef.external`; `labels` as the CA's own labels; `key_spec` (`algorithm`, or `kms_key_id` as `cloudKmsKeyVersionRef.external`); `key_usage` as base and extended key usages; `subject` and `subject_alt_name`; `is_ca`. Every default is fabric's, and unset means fabric's one CA, `test-ca` |
| `iam`, `iam_by_principals` | one `IAMBinding` per role on the pool, exact, the two merged as fabric merges them |
| `iam_bindings` | one `IAMBinding` per entry, exact, with its condition |
| `iam_bindings_additive` | one `IAMMember` per entry (additive), with its condition |
| `context` | refused: Bundt expressions (`${...}`) and `valueFrom` replace fabric's `$context` lookups |
| `allow_alpha` | not a fabric variable: no kind of this chart is alpha any more (`IAMMember` is `v1beta1` since config-connector-extensions `81c3f9b5`); kept so components that set it still render, it changes nothing |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the pool and the CAs (not the IAM) |

Refused:

- `ca_configs.<ca>.pem_ca_certificate` and `subordinate_config`:
  `PrivateCACertificateAuthority` has no `pemCaCertificate` or
  `subordinateConfig` field. A CA with `is_self_signed: false` is made as
  `SUBORDINATE` and waits for activation, which the chart cannot do.
- `ca_pool_config.create_pool.publishing_options.encoding_format: DER`:
  `PrivateCACAPool`'s `publishingOptions` has no `encodingFormat`, so the
  pool publishes PEM, the API's default (`PEM` is accepted).

Accepted with no effect: `ca_configs.<ca>.deletion_protection`,
`ignore_active_certificates_on_deletion` and `skip_grace_period` are
Terraform's own delete options, and Config Connector has none: deleting
the component deletes the CAs as Config Connector deletes them
(`deletion_policy: ABANDON` keeps them).

`lifetime`, which fabric leaves to the provider, is the provider's
default, `315360000s` (ten years), when not set: the kind requires it.
`gcs_bucket` is passed as `gcsBucketRef.external` as given (a bucket
name, as fabric's); the kind documents `projects/<p>/buckets/<b>`, and
which it accepts is not yet checked against the real API.

IAM is on the pool by its full name
(`projects/<project>/locations/<location>/caPools/<pool>`, or
`use_pool.id`), so the IAM objects can be deleted after the pool's.

## Outputs

| Output | From |
| --- | --- |
| `ca_pool_id` | the pool's full name: `use_pool.id`, or the rendered pool's |
| `ca_ids` | `{<ca>: projects/<p>/locations/<l>/caPools/<pool>/certificateAuthorities/<ca>}`, from the rendered CAs |
| `ca_chains` | `{<ca>: PEM chain}`, each CA's `status.pemCaCertificates` joined with newlines as fabric joins them; null until reported |
| `ca_pool`, `cas` | the pool's ID (null with `use_pool`, as fabric's is) and the CAs' IDs. fabric's are whole objects; a Package outputs the field a consumer takes (`docs/design.md`, "Outputs") |

## From Terraform

A `module "x" { source = ".../modules/certificate-authority-service" ... }`
block's arguments become the values unchanged, apart from `context` and
the refused fields.
The pool and the CAs keep their names, so they adopt what Terraform made
(`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/certificate-authority-service`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/certificate-authority-service)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=certificate-authority-service`: render cases `basic`
(fabric's README example), `existing` (`use_pool`), `full` (every
variable), `names` and `mock`, each validated against Config Connector's
CRDs and those in `crds/config-connector-extensions`, and the refusals in
`tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`): a DevOps pool, a self-signed root CA with fabric's defaults and a role on the pool (`iam`). On 2026-11-15 (offload VM `off-catalog-1115092825-18945`, config-connector-extensions `384d07c0`, whose fake cloud serves any `getIamPolicy` path) the pool, the CA and the `IAMBinding` became Ready, the outputs evaluated and all three were deleted. `use_pool`, KMS keys, subordinate CAs, `iam_bindings` (with conditions) and the `IAMMember`s are only render-tested.

Not applied live.

## Changes

- **0.2.0** (2026-11-15): `IAMMember` renders
  `iam.gcp.hfl.dev/v1beta1` without `allow_alpha` (config-connector-extensions
  `81c3f9b5`).
- **0.1.0** (2026-11-15): the port: the pool, its CAs and IAM on the pool.
