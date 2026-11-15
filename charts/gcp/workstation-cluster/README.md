# workstation-cluster

A Cloud Workstations cluster with its workstation configurations,
workstations and their IAM, as Config Connector and
config-connector-extensions objects. A port of Cloud Foundation Fabric's
[`modules/workstation-cluster`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/workstation-cluster)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-workstation-cluster` (`package.yaml`). Checked against Config
Connector v1.160.0 and config-connector-extensions at `384d07c0`.

## Example

```yaml
project_id: my-project
id: my-workstation-cluster
location: europe-west1
network_config:
  network: projects/my-project/global/networks/vpc
  subnetwork: projects/my-project/regions/europe-west1/subnetworks/subnet
workstation_configs:
  my-workstation-config:
    workstations:
      my-workstation:
        labels: {team: my-team}
```

renders the `WorkstationCluster` `my-workstation-cluster`, the
`WorkstationConfig` `my-workstation-config` in it and the `Workstation`
`my-workstation` in that (`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names.cluster` | not a fabric variable: the cluster's ID in full, used as given (no `namePrefix`) and checked (63 characters, a leading letter) (`docs/design.md`, "Names, labels and tags") |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the cluster's ID starts with `<namePrefix>-`. Configurations and workstations are named by their keys inside the cluster. Empty changes nothing |
| `labels` | Kubernetes labels on every object; an unprefixed key also becomes a Google Cloud label on the cluster (fabric's `labels`), and on every configuration and workstation |
| `tags` | cloud labels on every resource (Google Cloud labels) |
| `allow_alpha` | not a fabric variable: accept `v1alpha1` kinds, here config-connector-extensions' `IAMMember` for `iam_bindings_additive` (`docs/design.md`, "Alpha kinds") |
| `deletion_policy` | not a fabric variable: `DELETE` (default) or `ABANDON`; `PREVENT` is refused (`docs/design.md`, "Deletion") |
| `project_id`, `location`, `id`, `display_name`, `annotations` | the cluster's project, location, ID, display name and annotations |
| `network_config` | the cluster's `networkRef` and `subnetworkRef` (external) |
| `private_cluster_config` | `privateClusterConfig`: `enable_private_endpoint`, `allowed_projects` |
| `workstation_configs` | one `WorkstationConfig` per key, in the cluster: `display_name`, `annotations`, `labels`, `replica_zones`, `timeouts` (`idleTimeout`, `runningTimeout`, `<n>s`), `container` (`image`, `command`, `args`, `working_dir`, `env`, `run_as_user`, which must be a numeric user ID), `encryption_key`, `gce_instance` (`machine_type`, `service_account`, `service_account_scopes`, `pool_size`, `boot_disk_size_gb`, `tags`, `disable_public_ip_addresses`, `enable_nested_virtualization`, `enable_confidential_compute`, `shielded_instance_config`) and `persistent_directories`; each entry's `workstations`, one `Workstation` per key (`display_name`, `annotations`, `labels`); and IAM on both (below). `enable_audit_agent`, `max_workstations`, `gce_instance.accelerators`, `gce_instance.boost_configs` and a workstation's `env` are **refused**: Config Connector's `WorkstationConfig` and `Workstation` (v1.160.0) have no field for them |
| `iam`, `iam_bindings`, `iam_bindings_additive` (on a configuration or a workstation) | config-connector-extensions' kinds, since Config Connector's own IAM kinds cannot reference either resource: one `IAMBinding` (`iam.gcp.hfl.dev/v1beta1`) per role of `iam` and per entry of `iam_bindings`, exact as fabric's `iam_binding`s are; one `IAMMember` (`v1alpha1`, so `allow_alpha`) per entry of `iam_bindings_additive`. On the resource by its full name |
| `domain` | **refused**: Config Connector's `WorkstationCluster` (v1.160.0) has no `domainConfig` |
| `factories_config` | **refused**: fabric reads configurations from YAML files on disk; a component has no files, so each file's contents go under `workstation_configs.<file name>` |
| `context` | **refused**: Bundt's `${...}` expressions and `valueFrom` replace fabric's `$context` lookups |

## Outputs

| Output | From |
| --- | --- |
| `id` | the cluster's ID |
| `cluster_hostname`, `service_attachment_uri` | for a private cluster, `status.observedState.clusterHostname` and `serviceAttachmentUri`; null otherwise, or until reported |

These are fabric's declared outputs, selected (`docs/design.md`,
"Outputs"); `workstation_configs` and `workstations` (whole Terraform
objects) are not outputs.

## Differences from the module

- The refused fields above: a cluster's custom domain, a configuration's
  audit agent, its `max_usable_workstations`, accelerators and boost
  configurations, and a workstation's environment have no Config Connector
  field at v1.160.0.
- `container.run_as_user` is a string in fabric and an integer in Config
  Connector: a numeric user ID is carried, anything else is refused.
- fabric sends `disable_public_ip_addresses`, `enable_nested_virtualization`
  and the shielded-instance flags as `false` when unset; so does the chart.

## From Terraform

A `module "x" { source = ".../modules/workstation-cluster" ... }` block's
arguments become the values unchanged, except refused fields; files under
`factories_config.workstation_configs` become `workstation_configs`
entries, keyed by file name. To adopt an existing cluster, its
configurations and workstations, keep their IDs: Config Connector acquires
the resources of those names (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/workstation-cluster`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/workstation-cluster)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml) records
the repository, path, release, commit, licence and the date last synced, and
the module is listed in [`upstream/modules.yaml`](../../../upstream/modules.yaml)
with every other module the catalog tracks. To take a newer release, see
[`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=workstation-cluster`: render cases `basic`,
`private-cluster`, `custom-image` and `iam` (fabric's README examples),
`full` (every field the chart carries), `names` and `mock`, each validated
against Config Connector v1.160.0's CRD schemas and those in
`crds/config-connector-extensions` (`IAMBinding`, `IAMMember`), and the
refusals in `tests/*.fail.yaml`.

Applied to the fake cloud on 2026-11-15 on an offload VM (`task test-mock
MOCK_CHARTS="cloud-function-v1 workstation-cluster"`, run
`off-catalog-1115111105-4763`, at config-connector-extensions `384d07c0`,
whose fake cloud serves any `getIamPolicy` path): `tests/mock.values.yaml`,
a cluster, a configuration with an `IAMBinding` and one workstation: all 4
objects Ready, the outputs evaluated (`id`; the private-cluster outputs
null, as the cluster is public), all deleted. `iam_bindings`, the `v1alpha1`
`IAMMember`s and IAM on a workstation are render-tested only.

Not applied live.

## Changes

- **0.1.0** (2026-11-15): the port: the cluster, its workstation
  configurations and workstations, and IAM on both.
