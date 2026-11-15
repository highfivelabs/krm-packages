# looker-core

A Looker (Google Cloud core) instance: public, over Private Service
Access, or over Private Service Connect. A port of Cloud Foundation
Fabric's
[`modules/looker-core`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/looker-core)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-looker-core` (`package.yaml`). Config Connector has no kind for a
Looker instance; the chart uses config-connector-extensions'
`LookerInstance` (`looker.gcp.hfl.dev/v1alpha1`), at `069e4211`.

**The chart needs `allow_alpha: true`.** `LookerInstance` is `v1alpha1`,
so every render without it stops (`docs/design.md`, "Alpha kinds").

**The OAuth client secret comes from a Kubernetes Secret.**
`oauth_config.client_secret` is a reference, `{secretName, key}`, to a
Secret in the component's namespace (from External Secrets Operator,
`docs/secrets.md`), rendered as `oauthConfig.clientSecret.valueFrom.secretKeyRef`;
config-connector-extensions reads it and sends the value to Looker. The
secret itself, a string, stops the render. The builder's decision,
2026-11-15 (RECLOOK, `docs/design.md` decision 14).

## Example

```yaml
allow_alpha: true
project_id: my-project
region: europe-west1
name: looker
network_config:
  psa_config:
    network: projects/my-project/global/networks/my-network
oauth_config:
  client_id: 123-abc.apps.googleusercontent.com
  client_secret: {secretName: looker-oauth, key: client-secret}
platform_edition: LOOKER_CORE_ENTERPRISE_ANNUAL
```

renders the instance `looker` on `my-network` over Private Service
Access, private IP only, its OAuth client secret read from the Secret
`looker-oauth`, key `client-secret` (`tests/golden/psa.yaml`; fabric's
README example "Looker Core private instance with PSA"). The Secret, from
External Secrets Operator:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata: {name: looker-oauth, namespace: team-analytics}
spec:
  secretStoreRef: {kind: ClusterSecretStore, name: gcp-secret-manager}
  target: {name: looker-oauth}
  data:
    - {secretKey: client-secret, remoteRef: {key: looker-oauth-client-secret}}
```

## Values

The module's variables, with the same names, types and defaults. The
chart carries all 15; `oauth_config.client_secret` is a reference to a
Secret rather than the secret.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.instance` sets the instance's ID in full |
| `namePrefix` | not a fabric variable: `<namePrefix>-` in front of the instance's ID |
| `labels` | not a fabric variable: Kubernetes labels on the object (a Looker instance takes no Google Cloud labels) |
| `tags` | not a fabric variable: kept on the object's `metadata.labels` (the instance takes no cloud labels) |
| `allow_alpha` | not a fabric variable: required, `true` |
| `deletion_policy` | not a fabric variable: `DELETE` (default) or `ABANDON`; `PREVENT` is refused |
| `name`, `prefix` | the instance's ID (`resourceID`), `<prefix>-<name>` as fabric's; an empty `prefix` is refused, as in fabric |
| `project_id`, `region` | `projectRef` and `location` |
| `platform_edition` | `platformEdition` (default `LOOKER_CORE_TRIAL`); fabric's five editions only |
| `network_config` | exactly one of: `public: true` (`publicIpEnabled`); `psa_config` (`consumerNetwork`, `reservedRange` from `allocated_ip_range`, `privateIpEnabled` from `enable_private_ip`, default `true`, `publicIpEnabled` from `enable_public_ip`, default `false`); `psc_config` (`pscEnabled`, `pscConfig.allowedVpcs` and `serviceAttachments` with `localFqdn` and `targetServiceAttachmentUri`). `publicIpEnabled` and `pscEnabled` are always rendered, as fabric sets them |
| `oauth_config` | `oauthConfig.clientId` and `oauthConfig.clientSecret`. **`client_secret` is `{secretName, key}`**, rendered as `clientSecret.valueFrom.secretKeyRef` (`docs/secrets.md`); required, as in fabric |
| `admin_settings` | `adminSettings.allowedEmailDomains` |
| `custom_domain` | `customDomain.domain` |
| `controlled_egress` | `controlledEgressEnabled` (`enabled`, default `true`) and `controlledEgressConfig` (`marketplaceEnabled`, `egressFqdns`) |
| `encryption_config` | `encryptionConfig.kmsKeyName` |
| `fips_enabled`, `gemini_enabled` | `fipsEnabled`, `geminiEnabled` |
| `maintenance_config` | `maintenanceWindow` (`dayOfWeek` from `day`, default `SUNDAY`; `startTime`, default 23:00) and `denyMaintenancePeriod` (`startDate`, `endDate`, `time` from `start_time`, default 23:00) |
| `periodic_export_config` | `periodicExportConfig` (`kmsKey`, `gcsUri`, `startTime`, default 23:00) |

Fabric's validations stop the render with fabric's messages: exactly one
network mode, the platform edition, a maintenance window on the hour
(`0`-`23`) on a weekday name, and a non-empty `prefix`.

## Outputs

| Output | From |
| --- | --- |
| `id`, `instance_id` | the instance's `status.externalRef` (`projects/<p>/locations/<region>/instances/<id>`, Terraform's `id`); null until reported |
| `instance_name` | the instance's ID (`resourceID`) |
| `egress_public_ip`, `ingress_private_ip`, `ingress_public_ip`, `looker_uri`, `looker_version` | the instance's `status.observedState` (`egressPublicIp`, `ingressPrivateIp`, `ingressPublicIp`, `lookerUri`, `lookerVersion`); null until reported |

Not provided: `instance` (fabric's whole object; a Package outputs the
fields a consumer takes, `docs/design.md`, "Outputs"), and
`egress_service_attachments` and `looker_service_attachment`, the API's
read-only fields inside `pscConfig`, which config-connector-extensions'
`LookerInstance` does not report (it reports the top-level ones).

## Differences from the module

- **`client_secret` is a reference to a Kubernetes Secret**, not the
  secret, so it is never in the values, the rendered object or the
  component's inputs. A rotation is a change to the Secret;
  config-connector-extensions sends the new value at its next reconcile
  (its `status.inputVersion` follows the Secret's `resourceVersion`).
- **The maintenance window's start time is sent.** Fabric's
  `maintenance_window` block reads `start_times`, a misspelling of its own
  `start_time`, so a module call with a maintenance window fails to plan;
  the chart sends `start_time`.
- **Deletion works.** Fabric's README warns that Terraform cannot delete
  the instance (terraform-provider-google#19467); config-connector-extensions
  deletes it with the API's delete.

## From Terraform

A `module "x" { source = ".../modules/looker-core" ... }` block's arguments
become the values unchanged, with `allow_alpha: true`, except
`oauth_config.client_secret`: put the secret in a secret store, sync it
with External Secrets Operator, and replace the string with
`{secretName, key}`. The instance keeps its name, so it adopts what
Terraform made (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/looker-core`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/looker-core)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml) records
the repository, path, release, commit, licence and the date last synced, and
the module is listed in [`upstream/modules.yaml`](../../../upstream/modules.yaml)
with every other module the catalog tracks. To take a newer release, see
[`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=looker-core`: render cases `basic` (fabric's README
example "Simple example"), `psa` ("Looker Core private instance with
PSA"), `psc` ("Looker Core with PSC", with an egress service attachment),
`full` ("Looker Core full example", with every other variable set),
`names` and `mock`, each validated against config-connector-extensions'
CRD (`crds/config-connector-extensions`), and the refusals in
`tests/*.fail.yaml`, among them the secret as a string
(`secret-value.fail.yaml`) and a missing `oauth_config`.

Fake cloud (`task test-mock MOCK_CHARTS=looker-core`,
`tests/mock.values.yaml`): a public instance whose client secret comes
from a Secret that `hack/test-secret.sh` makes. On 2026-11-15 (offload VM
`off-catalog-1115230547-26084`, config-connector-extensions `069e4211`)
the instance became Ready with the client secret read from that Secret,
the outputs evaluated (`id`, `instance_id`, `instance_name`; the fake
reports no `lookerUri` or `lookerVersion`, so those were null), and the
instance was deleted: 0 failures.

Live: not run. A Looker instance is billed by the year and takes about an
hour to make, which the live rules exclude (`docs/design.md`, "Testing":
nothing billed while idle); config-connector-extensions' own live run of
the kind is pending for the same reason.
