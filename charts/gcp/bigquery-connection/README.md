# bigquery-connection

A BigQuery connection (Cloud SQL, AWS, Azure, Cloud Spanner, Cloud
resource or Spark) and IAM on it, as Config Connector objects. A port of
Cloud Foundation Fabric's
[`modules/bigquery-connection`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/bigquery-connection)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-bigquery-connection` (`package.yaml`). Checked against Config
Connector v1.160.0's `BigQueryConnectionConnection` and
config-connector-extensions' `IAMBinding` (`iam.gcp.hfl.dev/v1beta1`) and
`IAMMember` (`iam.gcp.hfl.dev/v1beta1`), at `81c3f9b5`: Config
Connector's own IAM kinds cannot reference a connection.

## Example

```yaml
project_id: my-project
location: EU
connection_id: my-connection
friendly_name: My connection
connection_config:
  cloud_sql:
    instance_id: my-project:europe-west1:my-instance
    database: my-database
    type: POSTGRES
    credential:
      secret_name: my-connection-credential   # a kubernetes.io/basic-auth Secret
iam:
  roles/bigquery.connectionUser: ["group:analysts@example.com"]
```

renders the `BigQueryConnectionConnection` `my-connection`, its Cloud SQL
credential read from the Kubernetes Secret `my-connection-credential`, and
an `IAMBinding` for the user role (`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.connection` sets the connection's ID in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the connection's ID (unless set under `names`) starts with `<namePrefix>-`; empty changes nothing |
| `labels`, `tags` | not fabric variables: Kubernetes labels on every object. A connection takes no Google Cloud labels |
| `project_id` | every object's project annotation, the connection's `projectRef` |
| `location` | the connection's `location` |
| `connection_id` | the connection's ID (`resourceID`) |
| `friendly_name`, `description` | the connection's `friendlyName` and `description` |
| `connection_config` | the connection's `cloudSQL` (instance, database and type; the credential from a Kubernetes Secret, below), `aws` (`accessRole.iamRoleID`), `azure` (customer tenant, federated application client), `cloudSpanner` (database, parallelism, Data Boost, maximum parallelism, database role), `cloudResource`, or `spark` (metastore service and Spark history server, as references by name) |
| `iam`, `iam_by_principals` | one `IAMBinding` per role, exact, the two merged as fabric merges them |
| `iam_bindings` | one `IAMBinding` per entry, exact, with its condition |
| `iam_bindings_additive` | one `IAMMember` per entry (additive), with its condition |
| `encryption_key` | refused (below) |
| `context` | refused: Bundt expressions (`${...}`) and `valueFrom` replace fabric's `$context` lookups |
| `allow_alpha` | not a fabric variable: no kind of this chart is alpha any more (`IAMMember` is `v1beta1` since config-connector-extensions `81c3f9b5`); kept so components that set it still render, it changes nothing |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the connection (not the IAM) |

Refused:

- `encryption_key`: `BigQueryConnectionConnection` has no `kmsKeyName`
  field.
- `connection_config.cloud_sql.credential`'s `username` and `password`:
  they would be in the rendered manifest. `BigQueryConnectionConnection`
  reads them from a Kubernetes Secret (`cloudSQL.credential.secretRef`),
  so the chart takes `credential.secret_name` instead (not a fabric
  field): a Secret of type `kubernetes.io/basic-auth` with `username` and
  `password`, in the component's namespace. A Cloud SQL connection
  without it stops the render.

IAM is on the connection by its full name
(`projects/<project>/locations/<location>/connections/<id>`), so the IAM
objects can be deleted after the connection's.

## Outputs

| Output | From |
| --- | --- |
| `connection_id` | the connection's ID, from the rendered connection |
| `location` | the connection's location |
| `description` | the connection's description, null when not set |
| `connection_config` | the inputs' `connection_config` (fabric reads it back from the connection, which holds what was set) |

## Differences from the module

- **The Cloud SQL credential** is a Kubernetes Secret, not a username and
  password (above).
- **`iam_bindings`**: fabric's resource reads `each.value.member`, which
  the variable does not have, so `iam_bindings` fails in Terraform; the
  chart renders each entry's `members` and condition.
- **Spark**: fabric's block needs both the metastore service and the
  history server; the chart sends whichever is set.
- **`$context` lookups** are refused (above).

## From Terraform

A `module "x" { source = ".../modules/bigquery-connection" ... }` block's
arguments become the values unchanged, apart from `context` and
`encryption_key` (refused) and a Cloud SQL credential (put it in a
Secret). The connection
keeps its ID, so it adopts what Terraform made (`docs/design.md`, "From
Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/bigquery-connection`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/bigquery-connection)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=bigquery-connection`: render cases `basic` (Cloud SQL
with a Secret), `spanner`, `aws`, `azure`, `full` (Spark and every IAM
variable), `names` and `mock`, each validated against Config Connector's
CRDs and those in `crds/config-connector-extensions`, and the refusals in
`tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`): a Cloud resource connection in `us` and a role on it (`iam`). On 2026-11-15 (offload VM `off-catalog-1115092825-18945`, config-connector-extensions `384d07c0`, whose fake cloud serves any `getIamPolicy` path) the connection and its `IAMBinding` became Ready, the outputs evaluated and both were deleted. The other connection types, `iam_by_principals`, `iam_bindings` (with conditions) and the `IAMMember`s are only render-tested; mockgcp reads no Cloud SQL Secret, so the Cloud SQL credential is unproven there.

Not applied live.

## Changes

- **0.2.0** (2026-11-15): `IAMMember` renders
  `iam.gcp.hfl.dev/v1beta1` without `allow_alpha` (config-connector-extensions
  `81c3f9b5`).
- **0.1.0** (2026-11-15): the port: the connection and IAM on it.
