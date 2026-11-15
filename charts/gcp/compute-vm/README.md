# compute-vm

A virtual machine, or a global instance template, with its disks, snapshot
and instance schedules, an unmanaged instance group, a service account,
instance IAM and IAP tunnel IAM, as Config Connector objects. A port of
Cloud Foundation Fabric's
[`modules/compute-vm`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/compute-vm)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-compute-vm` (`package.yaml`). Checked against Config Connector
v1.160.0 (`ComputeInstance`, `ComputeDisk`, `ComputeInstanceTemplate`,
`ComputeInstanceGroup`, `ComputeResourcePolicy`, `IAMServiceAccount`,
`IAMPartialPolicy`) and config-connector-extensions' kinds at its pin
`384d07c`: `ComputeDiskResourcePolicyAttachment` and
`ComputeRegionDiskResourcePolicyAttachment` (`compute.gcp.hfl.dev/v1beta1`;
Config Connector's own are `v1alpha1`), `IAMBinding`
(`iam.gcp.hfl.dev/v1beta1`) and `IAMMember` (`v1alpha1`, behind
`allow_alpha`) on the IAP tunnel instance. A managed group of these VMs is
`compute-mig`, which takes the template this chart makes by its
`template_name` output.

## Example

```yaml
project_id: my-project
zone: europe-west8-b
name: test
network_interfaces:
  - network: projects/my-project/global/networks/vpc
    subnetwork: projects/my-project/regions/europe-west8/subnetworks/subnet
```

renders the `ComputeInstance` `test`, an `e2-micro` booting
`debian-cloud/debian-13` from a 10 GB `pd-balanced` disk, running as the
Compute Engine default service account with fabric's three default scopes
(`tests/golden/simple.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names.instance`, `names.group`, `names.instanceSchedule`, `names.serviceAccount`, `names.template` | not fabric variables: the instance's name, the unmanaged group's and the instance schedule policy's (fabric gives both the instance's name; unset, they take the instance's name as set), the created account's ID (30 characters), and the template's base name (54 characters, before the chart's `-<hash>`), in full, used as given (no `namePrefix`) (`docs/design.md`, "Names, labels and tags") |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the instance (and so its disks, `<instance>-<key>`, and snapshot schedules), the account and the template start with `<namePrefix>-`, and so does a disk's own `name`; empty changes nothing |
| `labels` | fabric's instance labels; as every chart's `labels`, Kubernetes labels on every object, an unprefixed key also a Google Cloud label on every resource that takes one; and, unprefixed, merged into the boot disk's `initializeParams.labels`, as fabric merges them |
| `tags` | a list is fabric's network tags, `tags` on the instance or template; a map is every chart's `tags`, cloud labels on every resource (and on the boot disk) |
| `project_id`, `zone` | every object's project annotation (and `projectRef` where the kind has one); the instance's `zone`, the region (fabric's `local._region`) of the policies, regional disks and template |
| `name`, `description`, `hostname`, `machine_type`, `min_cpu_platform`, `can_ip_forward`, `enable_display`, `metadata`, `metadata_startup_script` | `resourceID`, `description`, `hostname`, `machineType`, `minCpuPlatform`, `canIpForward`, `enableDisplay` (instance only, as fabric's), `metadata` (Config Connector's `{key, value}` list, in key order), `metadataStartupScript` |
| `boot_disk` | `bootDisk`: `initializeParams` (`size`, `type`, the image as `sourceImageRef.external`, the immutable tags, labels `disk_name: boot`, `disk_type`), or `sourceDiskRef.external` (`source.snapshot` or `source.attach`, as fabric's coalesce), `autoDelete`; `use_independent_disk` makes a `ComputeDisk` `<instance>-boot` (or its `name`) with the image, type, size, IOPS and throughput, attached with `autoDelete: false`. On a template, the boot entry of `disk` |
| `attached_disks` | in fabric's order (`position`, else the key): an `attachedDisk` each (`deviceName`, `mode`, `sourceDiskRef`); a disk without `source.attach` is a `ComputeDisk` (`<instance>-<key>` or its `name`; image, snapshot, type, size, IOPS, throughput, labels `disk_name` and `disk_type`), regional (`location` the region, `replicaZones` the zone and `replica_zone`) when `replica_zone` is set. On a template, a `PERSISTENT` entry of `disk` each |
| `encryption` | `kms_key_self_link`: the boot disk's `kmsKeyRef` and every made disk's `diskEncryptionKey.kmsKeyRef` (on a template, every disk's), as fabric |
| `confidential_compute`, `gpu`, `shielded_config`, `machine_features_config`, `network_performance_tier`, `scratch_disks` | `confidentialInstanceConfig.enableConfidentialCompute`, `guestAccelerator`, `shieldedInstanceConfig`, `advancedMachineFeatures` (nested virtualization, threads per core, visible cores), `networkPerformanceConfig`, `scratchDisk` (on a template, `SCRATCH` entries of `disk`) |
| `network_interfaces` | `networkInterface`: network and subnetwork as `external` references, `networkIp`, `nicType`, `stackType`, `queueCount`, `internalIpv6PrefixLength`, `accessConfig` (with `nat` or a tier; `natIpRef.external` the external address), `ipv6AccessConfig` (Premium) with `external_ipv6`, `aliasIpRange` |
| `network_attached_interfaces` | on a template, a `networkInterface` each with `networkAttachment` |
| `scheduling_config`, `lifecycle_config` | `scheduling`, with fabric's defaults: `automaticRestart` false for Spot, `onHostMaintenance` `TERMINATE` for Spot, confidential or GPU VMs, `instanceTerminationAction` (`STOP` by default) for Spot or a maximum run duration, `preemptible` for Spot, node affinities (`IN`, or `NOT_IN` with `in: false`); `deletionProtection`; `allow_stopping_for_update` (true by default) is the annotation `cnrm.cloud.google.com/allow-stopping-for-update` |
| `service_account` | `{}`: the Compute Engine default account; `email`: that account; `auto_create`: an `IAMServiceAccount` `tf-vm-<name>` the instance references by name; null: none. Scopes default to fabric's |
| `resource_policies`, `instance_schedule` | `resourcePolicies`: each ID as `external`, then the schedule: a `ComputeResourcePolicy` with `instanceSchedulePolicy`, named as the instance |
| `snapshot_schedules` | a `ComputeResourcePolicy` `<instance>-<key>` each, with `snapshotSchedulePolicy` |
| `boot_disk.snapshot_schedule`, `attached_disks.*.snapshot_schedule` | a `ComputeDiskResourcePolicyAttachment` (a regional disk's: `ComputeRegionDiskResourcePolicyAttachment`) per disk and schedule: a key of `snapshot_schedules` names the chart's policy, anything else an existing policy, as fabric's `try()`; the boot disk Compute Engine makes is the instance's name, as fabric's |
| `group` | `named_ports` (or `{}`): a `ComputeInstanceGroup` named as the instance, holding it, on the first interface's network |
| `iam` | one `IAMPartialPolicy` on the instance (`docs/design.md`, "IAM") |
| `iap_tunnel_iam`, `iap_tunnel_iam_bindings` | one `IAMBinding` per role, or per entry, on the IAP tunnel instance (`projects/<p>/iap_tunnel/zones/<z>/instances/<name>`), exact per role as fabric's `google_iap_tunnel_instance_iam_binding` |
| `iap_tunnel_iam_bindings_additive` | one `IAMMember` (`v1alpha1`) per entry: needs `allow_alpha: true` |
| `tag_bindings_immutable` | the instance's `params.resourceManagerTags` and the boot disk's `initializeParams.resourceManagerTags` |
| `create_template` | `{}`: a global `ComputeInstanceTemplate` instead of an instance ("Templates") |
| `project_number` | nothing: fabric uses it only in tag bindings, refused here |
| `deletion_policy`, `allow_alpha` | not fabric variables; `ABANDON` keeps every resource; `allow_alpha` accepts `IAMMember` |

Refused (`upstream.yaml`), each naming its gap:

- `context`: Bundt's `${...}` expressions and `valueFrom` replace it.
- `kms_autokeys`: a key handle's key is known only after apply, and no
  disk or instance can reference a `KMSKeyHandle` (gap 8); give the key in
  `encryption.kms_key_self_link`.
- `tag_bindings`, `network_tag_bindings`: a VM's and its disks' bindings
  name their numeric IDs, known only after apply, and
  `TagsLocationTagBinding` cannot reference a `ComputeInstance` or
  `ComputeDisk` (gap 8). `tag_bindings_immutable` sets tags at creation.
- `tpu_config`: `TPUVirtualMachine` and `TPUQueuedResource` are alpha and
  not ported yet, even with `allow_alpha`.
- `create_template.regional`: config-connector-extensions'
  `ComputeRegionInstanceTemplate` is `v1alpha1` and not ported yet (its
  schema is the API's, a second mapping of every field); use a global
  template.
- `encryption.disk_encryption_key_raw`: a raw key given here would be in
  the rendered manifest; `ComputeInstance` and `ComputeDisk` take it from a
  Kubernetes Secret (`valueFrom`), which the chart does not take yet (the
  SECV question).
- `group.membership`: `ComputeInstanceGroup` owns its group's whole
  instance list, and there is no membership kind.
- `lifecycle_config.key_revocation_action_type`, `.graceful_shutdown`;
  `machine_features_config.enable_turbo_mode`, `.enable_uefi_networking`,
  `.performance_monitoring_unit`; `boot_disk.architecture`,
  `.force_attach`, `attached_disks.*.force_attach`: no such fields.
- Hyperdisk settings where the kind has no field: on an instance's boot
  disk without `use_independent_disk`; `storage_pool` on any disk the chart
  makes; `provisioned_throughput` on a template.
- `network_attached_interfaces` on an instance (`ComputeInstance` has no
  `networkAttachment`; a template has).
- With `create_template`: `tag_bindings_immutable` (no
  `resourceManagerTags`), `iam` and the IAP tunnel IAM (no instance), the
  snapshot schedules of disks (no disk).

## Templates

fabric names a template with `name_prefix = "<name>-"` and
`create_before_destroy`, so Terraform makes a new template, under a new
generated name, for every change: a template cannot be changed in place.
The chart does the same deterministically: the template's `resourceID`,
and its object's name, end with 8 hex characters of the sha256 of its spec
and labels. A changed template is a new object, which Config Connector
creates, and Bundt prunes the old one; Compute Engine refuses to delete it
while a group still uses it, and Config Connector retries until the group
has moved to the new one (by `template_name`). The base name is
`names.template`, else `name` with the `namePrefix`, at most 54
characters.

## Outputs

| Output | Value |
| --- | --- |
| `id` | `projects/<project>/zones/<zone>/instances/<name>`, the provider's ID |
| `self_link` | the instance's `status.selfLink` |
| `internal_ip`, `internal_ips` | the first interface's, and every interface's, `networkIp`, as Config Connector writes them back into the spec (null, or empty, until it has) |
| `external_ip` | with `nat` on the first interface, its `accessConfig[0].natIpRef.external` as Config Connector writes it back, else null |
| `login_command` | `gcloud compute ssh --project <project> --zone <zone> <name>` |
| `service_account_email`, `service_account_iam_email` | the created account's email, or `service_account.email`, or null (the default account, or none), and as an IAM member |
| `template_name` | the template's name, with `create_template` |

The instance outputs are null with `create_template`, and `template_name`
without it. fabric's `group`, `instance`, `service_account` and `template`
are whole Terraform objects, which a Package does not output
(`docs/design.md`, "Outputs").

## Differences from the module

- The template's generated name is `<name>-<hash of the spec>`, not
  Terraform's timestamp ("Templates").
- `labels` and `tags` are every chart's values too: they label every
  object in Kubernetes, and a map of `tags` adds cloud labels.
- fabric reads `boot_disk.source.disk`, `encryption.encrypt_boot` and
  `instance_schedule.active` nowhere, and neither does the chart.

## From Terraform

A `module "x" { source = ".../modules/compute-vm" ... }` block's arguments
become the values unchanged, except refused fields
(`examples/compute-vm/`). To adopt an existing VM, keep its name and zone
and set `deletion_policy: ABANDON` for the first apply (`docs/design.md`,
"From Terraform"). A template adopted from Terraform keeps its generated
name only if `names.template` and the hash give it, which they will not:
Bundt makes a new template and the group moves to it.

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/compute-vm`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/compute-vm)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml) records
the repository, path, release, commit, licence and the date last synced, and
the module is listed in [`upstream/modules.yaml`](../../../upstream/modules.yaml)
with every other module the catalog tracks. To take a newer release, see
[`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=compute-vm`: render cases `simple` (fabric's first README
example), `full` (every field the chart carries on an instance),
`template`, `alpha` (the additive IAP tunnel IAM), `names` and `mock`, each
validated against Config Connector v1.160.0's CRD schemas and
config-connector-extensions' CRDs, and the refusals in `tests/*.fail.yaml`.

Fake cloud: `task test-mock MOCK_CHARTS=compute-vm`:
`tests/mock.values.yaml`, an instance with a created service account, an
attached disk with a snapshot schedule attached to it, an instance schedule
and an unmanaged group, on the network in `tests/mock.needs.yaml`. Result:
pending. Not in the fake-cloud case, render-tested only: instance IAM and
IAP tunnel IAM (mockgcp has neither), a snapshot schedule on the boot disk
(mockgcp makes no boot disk with an instance), a regional disk, and the
template.

## Changes

- **0.1.0** (2026-11-15): the port: the instance or a global template,
  its disks, schedules, group, service account and IAM.
