# analytics-hub

A BigQuery Analytics Hub data exchange, its listings and IAM on both, as
Config Connector objects. A port of Cloud Foundation Fabric's
[`modules/analytics-hub`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/analytics-hub)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-analytics-hub` (`package.yaml`). Checked against Config Connector
v1.160.0's `BigQueryAnalyticsHubDataExchange` and
`BigQueryAnalyticsHubListing`, and config-connector-extensions'
`IAMBinding` (`iam.gcp.hfl.dev/v1beta1`) and `IAMMember`
(`iam.gcp.hfl.dev/v1beta1`), at `81c3f9b5`: Config Connector's own IAM
kinds cannot reference either resource.

## Example

```yaml
project_id: my-project
region: europe-west1
prefix: test
name: exchange
primary_contact: exchange-owner-group@domain.com
description: Test Data Exchange
iam:
  roles/analyticshub.viewer: ["group:viewer@domain.com"]
listings:
  listing_id:
    bigquery_dataset: projects/my-project/datasets/my_dataset
    iam:
      roles/analyticshub.subscriber: ["group:subscriber@domain.com"]
```

renders the data exchange `test_exchange`, the listing `listing_id`
sharing the dataset, and an `IAMBinding` for each role on each
(`tests/golden/basic.yaml`, fabric's README example).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.data_exchange` sets the exchange's ID in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the exchange's ID (unless set under `names`) starts with `<namePrefix>_`, its hyphens as underscores (an exchange ID has no hyphens). Listings keep their keys: they live in the exchange |
| `labels`, `tags` | not fabric variables: Kubernetes labels on every object; Analytics Hub resources take no Google Cloud labels |
| `project_id` | every object's project annotation and `projectRef` |
| `region` | every object's `location` |
| `name`, `prefix` | the exchange's ID and display name, `<prefix>_<name>` as fabric makes them (`<name>` without a prefix). Letters, digits and underscores only |
| `description`, `documentation`, `primary_contact` | the exchange's `description`, `documentation`, `primaryContact` |
| `listings` | one `BigQueryAnalyticsHubListing` per key, its ID and display name the key, in the exchange: `bigquery_dataset` as the shared dataset, `description`, `documentation`, `categories`, `primary_contact`, `request_access`, `restricted_export_config` as `restrictedExportPolicy`, and `iam` as one `IAMBinding` per role on the listing. `data_provider` and `publisher` are accepted and not sent, as fabric does; `icon` is refused (below) |
| `iam`, `iam_by_principals`, `iam_bindings` | one `IAMBinding` per role on the exchange, exact: the three merged by role as fabric merges them (an `iam_bindings` key makes no binding of its own) |
| `iam_bindings_additive` | one `IAMMember` per entry on the exchange (additive) |
| `factories_config` | refused (below) |
| `icon` | refused (below) |
| `allow_alpha` | not a fabric variable: no kind of this chart is alpha any more (`IAMMember` is `v1beta1` since config-connector-extensions `81c3f9b5`); kept so components that set it still render, it changes nothing |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the exchange and its listings (not the IAM) |

Refused:

- `icon`, and a listing's `icon`: neither
  `BigQueryAnalyticsHubDataExchange` nor `BigQueryAnalyticsHubListing` has
  an icon field.
- `factories_config`: fabric reads listings from YAML files on disk, and
  a component has no files; put the listings in `listings`.

IAM is on the exchange and listings by their full names
(`projects/<project>/locations/<region>/dataExchanges/<id>[/listings/<listing>]`),
so the IAM objects can be deleted after theirs.

## Outputs

| Output | From |
| --- | --- |
| `data_exchange_id` | the exchange's ID, from the rendered exchange |
| `data_listings` | `{<listing>: <dataset>}`, the dataset each listing shares, from the rendered listings. fabric's value is the listing's `bigquery_dataset` block, whose field is the dataset |

## Differences from the module

- **`data_provider` and `publisher`** on a listing are accepted and not
  sent: fabric's variable has them and its resource does not send them.
- **Factories** are refused (above).

## From Terraform

A `module "x" { source = ".../modules/analytics-hub" ... }` block's
arguments become the values unchanged, apart from `icon` and
`factories_config` (refused; a factory's listing files become entries of
`listings`). The exchange and listings keep their IDs, so they adopt what Terraform made
(`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/analytics-hub`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/analytics-hub)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=analytics-hub`: render cases `basic` (fabric's README
example), `full` (every variable), `names` and `mock`, each validated
against Config Connector's CRDs and those in
`crds/config-connector-extensions`, and the refusals in
`tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`, with the shared
dataset in `tests/mock.needs.yaml`): an exchange in `us`, a listing sharing a dataset, and a role on each. On 2026-11-15 (offload VM `off-catalog-1115092825-18945`, config-connector-extensions `384d07c0`, whose fake cloud serves any `getIamPolicy` path) the exchange, the listing and both `IAMBinding`s became Ready, the outputs evaluated and all four were deleted (the dataset too). The `IAMMember`s are only render-tested.

Not applied live.

## Changes

- **0.2.0** (2026-11-15): `IAMMember` renders
  `iam.gcp.hfl.dev/v1beta1` without `allow_alpha` (config-connector-extensions
  `81c3f9b5`).
- **0.1.0** (2026-11-15): the port: the exchange, its listings and IAM on
  both.
