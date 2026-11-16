# net-lb-proxy-int

A regional internal proxy Network Load Balancer (an L4 TCP proxy ILB):
forwarding rules in a subnet, a regional TCP proxy, a regional backend
service and health check, optional unmanaged instance groups, network
endpoint groups (zonal, hybrid, internet, Private Service Connect) and a
PSC service attachment, as Config Connector objects. A port of Cloud
Foundation Fabric's
[`modules/net-lb-proxy-int`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/net-lb-proxy-int)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-net-lb-proxy-int` (`package.yaml`). Checked against Config Connector
v1.160.0 (`ComputeForwardingRule`, `ComputeTargetTCPProxy`,
`ComputeBackendService`, `ComputeHealthCheck`, `ComputeInstanceGroup`,
`ComputeNetworkEndpointGroup`, `ComputeRegionNetworkEndpointGroup`,
`ComputeServiceAttachment`) and config-connector-extensions'
`ComputeNetworkEndpoint` (`compute.gcp.hfl.dev/v1beta1`) and
`ComputeRegionNetworkEndpoint` (`v1beta1`) at `81c3f9b5`. Written after
`net-lb-proxy-int-cross-region`.

Regional internal proxy load balancers need a proxy-only subnet with
purpose `REGIONAL_MANAGED_PROXY` in the region (net-vpc's
`subnets_proxy_only`); the chart does not make it.

## Example

```yaml
name: ilb-test
project_id: my-project
region: europe-west1
backend_service_config:
  backends:
    - group: projects/my-project/zones/europe-west1-b/instanceGroups/my-ig
vpc_config:
  network: projects/my-project/global/networks/my-vpc
  subnetwork: projects/my-project/regions/europe-west1/subnetworks/my-subnet
```

renders the forwarding rule `ilb-test` (port 80, global access) in
`europe-west1`, the regional TCP proxy and backend service `ilb-test`,
and its regional TCP health check `ilb-test` on the serving port
(`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.backendService`, `names.healthCheck` and `names.targetProxy` set those cloud names in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, every name the chart derives or is given in an entry starts with `<namePrefix>-`; empty changes nothing |
| `labels` | fabric's labels: Kubernetes labels on every object; an unprefixed key also becomes a Google Cloud label on the forwarding rules (the only resources here with labels) |
| `tags` | not a fabric variable: cloud labels on the forwarding rules |
| `name` | the TCP proxy's and service attachment's name, and the base of every derived name |
| `project_id` | every object's project, unless an entry sets its own |
| `region` | the region of every object but the zonal NEGs and groups |
| `vpc_config` | `network` and `subnetwork` of the forwarding rules, and of zonal NEGs without their own |
| `forwarding_rules_config` | one regional `ComputeForwardingRule` (`INTERNAL_MANAGED`, TCP) per key, named `name` for the key `""` and `<name>-<key>` otherwise unless it has a name: address (an IP, or an address's ID or self link), IPv6 (only without an address, as fabric), description (else the chart's), global access (`true` unless `false`), port (80). `protocol` is accepted and unused, as in fabric. `null` (the default) is fabric's default, `{"": {}}` |
| `description` | the forwarding rules', proxy's, NEGs' and service attachment's description |
| `backend_service_config` | the regional `ComputeBackendService` (`INTERNAL_MANAGED`, `TCP`): name, description, cookie TTL, draining timeout, port name, session affinity (`NONE` or `CLIENT_IP`), timeout, logging, and the backends (group, balancing mode, `UTILIZATION` by default, capacity scaler, description, `max_connections` per group, endpoint or instance, max utilization). `log_config.optional_mode` and `optional_fields` are refused (below). `health_checks` and `project_id` are accepted and unused, as in fabric |
| `health_check`, `health_check_config` | an existing health check, or a regional `ComputeHealthCheck` of one type (TCP on the serving port by default; its `project_id` is unused, as in fabric). None when a backend is one of the chart's PSC NEGs, or neither is set, as fabric |
| `group_configs` | one unmanaged `ComputeInstanceGroup` per key, named `<name>-<key>` unless it has a name, in its own project if it sets one |
| `neg_configs` | per key, one of: `gce`, a zonal `ComputeNetworkEndpointGroup` (`GCE_VM_IP_PORT`; network and subnetwork the entry's or `vpc_config`'s) with one `ComputeNetworkEndpoint` per endpoint; `hybrid`, the same of type `NON_GCP_PRIVATE_IP_PORT`; `internet`, a `ComputeRegionNetworkEndpointGroup` (`INTERNET_FQDN_PORT`, or `INTERNET_IP_PORT` with `use_fqdn: false`) on `vpc_config.network` with one `ComputeRegionNetworkEndpoint` per endpoint; `psc`, a `ComputeRegionNetworkEndpointGroup` to a PSC target service. Named `<name>-<key>`; zonal ones in the entry's project if it sets one, the others in the chart's, as fabric |
| `service_attachment` | one `ComputeServiceAttachment` on the forwarding rule of the key `forwarding_rule` (`""` by default), named `name`: NAT subnets, automatic or manual acceptance, accept and reject lists, proxy protocol. `domain_name` and `reconcile_connections` are refused (below) |
| `context` | refused: Bundt expressions (`${...}`) and `valueFrom` replace fabric's `$context` lookups |
| `allow_alpha` | not a fabric variable, and no longer used: no kind of this chart is alpha (internet NEG endpoints' `ComputeRegionNetworkEndpoint` is `v1beta1` since config-connector-extensions `81c3f9b5`). It changes nothing and is kept so components that set it still render |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps every resource |

Refused, by name:

- `backend_service_config.log_config.optional_mode` and `optional_fields`:
  `ComputeBackendService`'s `logConfig` has neither.
- `neg_configs.<key>.psc.producer_port`:
  `ComputeRegionNetworkEndpointGroup` has no `pscData`.
- `service_attachment.domain_name` and `reconcile_connections`:
  `ComputeServiceAttachment` has neither field.

fabric's validations are kept: one health check type, one NEG type per
entry, the session affinity and the balancing modes; a service
attachment's `forwarding_rule` must be a `forwarding_rules_config` key.
Name lengths: with a `namePrefix`, a derived name over 63 characters
stops the render; a name set under `names` is always checked.

## Outputs

| Output | From |
| --- | --- |
| `address` | each forwarding rule's `spec.ipAddress.ip`, by rule name as fabric keys it: the address given, or the one Config Connector writes back; else null |
| `backend_service_id` | `projects/<project>/regions/<region>/backendServices/<name>` |
| `backend_service_self_link` | the backend service's `status.selfLink` |
| `group_self_links` | each group's `status.selfLink`, by key |
| `health_check_id`, `health_check_self_link` | the auto-created health check's ID and `status.selfLink`, or null |
| `ids` | `projects/<project>/regions/<region>/forwardingRules/<name>`, by key |
| `neg_ids` | `projects/<project>/zones/<zone>/networkEndpointGroups/<name>`, by key, of the zonal NEGs |
| `service_attachment_id` | `projects/<project>/regions/<region>/serviceAttachments/<name>`, or null |

Not outputs: fabric's whole objects (`backend_service`,
`forwarding_rules`, `groups`, `health_check`; `docs/design.md`,
"Outputs").

## Differences from the module

- **Regional NEGs as backends** (PSC, internet) are referenced by their
  ID, since a backend's NEG reference by name is to a zonal group;
  Config Connector retries the backend service until the group exists.
- **Map defaults.** Helm merges a map given in the values into the
  chart's default, where Terraform replaces it, so
  `forwarding_rules_config` defaults to null and the chart applies
  fabric's default; `health_check_config` keeps
  net-lb-proxy-int-cross-region's handling of the merged default TCP
  check.
- **Instance groups and NEGs together**, **duplicate backends**: as in
  `net-lb-proxy-int-cross-region`.

## From Terraform

A `module "x" { source = ".../modules/net-lb-proxy-int" ... }` block's
arguments become the values unchanged, apart from `context` (refused:
write the values the lookups stand for). Every resource keeps its name,
so it adopts what Terraform made (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/net-lb-proxy-int`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/net-lb-proxy-int)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=net-lb-proxy-int`: render cases from fabric's README,
`basic` ("Minimal Example"), `forwarding-rules` (with an existing health
check and a service attachment), `negs` (zonal, hybrid, internet with
endpoints), `psc-neg`; and `names` and `mock`. Each is validated
against Config Connector v1.160.0's CRD schemas and
config-connector-extensions' CRDs; the refusals are in
`tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`, with the network
and subnet from `tests/mock.needs.yaml`). On 2026-11-15 (offload VM `off-catalog-1115102556-2238`,
config-connector-extensions `384d07c0`) all 6 objects became Ready, the
outputs evaluated and all 6 were deleted. The
PSC and internet NEGs and the service attachment are only render-tested.

## Changes

- **0.2.0** (2026-11-15): internet NEG endpoints render config-connector-extensions'
  `v1beta1` `ComputeRegionNetworkEndpoint` and need no `allow_alpha` (now
  a no-op).
- **0.1.0** (2026-11-15): the port.
