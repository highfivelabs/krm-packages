# net-lb-app-ext-regional

A regional external Application Load Balancer (`EXTERNAL_MANAGED`): a
forwarding rule on a network, an HTTP or HTTPS target proxy with its SSL
certificates (existing, or created from a PEM whose private key is a
Kubernetes Secret), a URL map, regional backend services and health
checks, optional unmanaged instance groups and network endpoint groups
(zonal, hybrid, Cloud Run, Cloud Functions, internet, Private Service
Connect), as Config Connector objects. A port of Cloud Foundation
Fabric's
[`modules/net-lb-app-ext-regional`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/net-lb-app-ext-regional)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-net-lb-app-ext-regional` (`package.yaml`). Checked against Config
Connector v1.160.0 (`ComputeForwardingRule`, `ComputeTargetHTTPProxy`,
`ComputeTargetHTTPSProxy`, `ComputeSSLCertificate`, `ComputeURLMap`,
`ComputeBackendService`, `ComputeHealthCheck`, `ComputeInstanceGroup`,
`ComputeNetworkEndpointGroup`, `ComputeRegionNetworkEndpointGroup`) and
config-connector-extensions' `ComputeNetworkEndpoint`
(`compute.gcp.hfl.dev/v1beta1`) and `ComputeRegionNetworkEndpoint`
(`v1beta1`) at `81c3f9b5`.

Regional external application load balancers need a proxy-only subnet
with purpose `REGIONAL_MANAGED_PROXY` in the region (net-vpc's
`subnets_proxy_only`); the chart does not make it.

The URL map, backend services, health checks, groups, NEGs and SSL
certificates are rendered by `templates/_netlbapp.tpl`, which
`net-lb-app-ext` and `net-lb-app-int` carry identical copies of; each
chart's `netlbapp.cfg` says what differs.

## Example

```yaml
name: glb-test-0
project_id: my-project
region: europe-west8
backend_service_configs:
  default:
    backends:
      - group: projects/my-project/zones/europe-west8-b/instanceGroups/my-ig-b
vpc_config:
  network: projects/my-project/global/networks/my-vpc
```

renders the forwarding rule `glb-test-0` (HTTP, port 80, standard tier)
in `europe-west8`, the regional target HTTP proxy and URL map
`glb-test-0`, the regional backend service `glb-test-0-default` and its
regional HTTP health check `glb-test-0-default` (`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.targetProxy` and `names.urlMap` set those cloud names in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, every name the chart derives or is given in an entry starts with `<namePrefix>-`; empty changes nothing |
| `labels` | fabric's labels: Kubernetes labels on every object; an unprefixed key also becomes a Google Cloud label on the forwarding rule (the only resource here with labels) |
| `tags` | not a fabric variable: cloud labels on the forwarding rule |
| `name` | the forwarding rule's, proxy's and URL map's name, and the base of every derived name |
| `project_id` | every object's project, unless a backend service, health check, group or serverless NEG sets its own |
| `region` | the region of the forwarding rule, proxy, URL map, backend services, health checks and certificates |
| `vpc_config` | `network` of the forwarding rule |
| `address` | the forwarding rule's address: an IP, or an address's ID or self link |
| `network_tier_standard` | `STANDARD` (the default) or, with `false`, `PREMIUM` |
| `protocol` | `HTTP` (a `ComputeTargetHTTPProxy`) or `HTTPS` (a `ComputeTargetHTTPSProxy`), and the backend services' protocol unless they set one |
| `ports` | the forwarding rule's port for HTTP (at most one; 80 by default); HTTPS is 443, as fabric |
| `description` | the forwarding rule's and NEGs' description (a NEG's own first) |
| `http_proxy_config`, `https_proxy_config` | the proxy's name, description and keep-alive timeout; for HTTPS also Certificate Manager certificates, the server TLS policy and the SSL policy. `certificate_map` and `quic_override` are accepted and unused, as in fabric |
| `ssl_certificates` | the HTTPS proxy's certificates: `certificate_ids` (existing, by ID) and `create_configs`, one regional `ComputeSSLCertificate` per key named `<name>-<key>` unless it has a name, the certificate (public) as given and the private key a reference to a Kubernetes Secret, `{secretName, key}` (`docs/secrets.md`) |
| `urlmap_config` | the regional `ComputeURLMap`: description, default service, default redirect, default route action, header action, host rules, path matchers (default service, default redirect, default route action, header action, path rules, route rules with their match rules), tests. A service is the key of one of the chart's backend services, else an existing one's ID or self link. `null` (the default) is fabric's default, `{default_service: default}` |
| `backend_service_configs` | one regional `ComputeBackendService` (`EXTERNAL_MANAGED`) per key, named `<name>-<key>` unless it has a name: description, cookie TTL, draining timeout, CDN and its policy, health checks (keys of the chart's or existing IDs; `["default"]` unless set, `[]` for none), locality LB policy, logging, port name (`lower(protocol)` unless set, as fabric), protocol, security policy, session affinity, timeout, circuit breakers, consistent hash, IAP (the client secret a Secret reference), outlier detection and the backends (with `max_connections` and `max_rate`). `log_config.optional_mode` and `optional_fields` are refused (below) |
| `health_check_configs` | one regional `ComputeHealthCheck` per key, named `<name>-<key>` unless it has a name, of one type. `null` (the default) is fabric's default; `{}` makes none |
| `group_configs` | one unmanaged `ComputeInstanceGroup` per key |
| `neg_configs` | per key, one of: `gce`, a zonal `ComputeNetworkEndpointGroup` (`GCE_VM_IP_PORT`) with one `ComputeNetworkEndpoint` per endpoint; `hybrid`, the same of type `NON_GCP_PRIVATE_IP_PORT`; `cloudrun` or `cloudfunction`, a serverless `ComputeRegionNetworkEndpointGroup` in the entry's project; `internet`, a `ComputeRegionNetworkEndpointGroup` on the entry's network, `INTERNET_FQDN_PORT` when an endpoint has an `fqdn` (else `INTERNET_IP_PORT`), with one `ComputeRegionNetworkEndpoint` per endpoint; `psc`, a `ComputeRegionNetworkEndpointGroup` to a PSC target service. Named `<name>-<key>` |
| `context` | refused: Bundt expressions (`${...}`) and `valueFrom` replace fabric's `$context` lookups |
| `allow_alpha` | not a fabric variable, and no longer used: no kind of this chart is alpha (internet NEG endpoints' `ComputeRegionNetworkEndpoint` is `v1beta1` since config-connector-extensions `81c3f9b5`). It changes nothing and is kept so components that set it still render |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps every resource |

Refused, by name:

- `backend_service_configs.<key>.log_config.optional_mode` and
  `optional_fields`: `ComputeBackendService`'s `logConfig` has neither.
- `ssl_certificates.create_configs.<key>.private_key` and
  `backend_service_configs.<key>.iap_config.oauth2_client_secret` given as
  a value: they take a reference to a Kubernetes Secret,
  `{secretName, key}`, rendered as Config Connector's
  `valueFrom.secretKeyRef` (`docs/secrets.md`).

fabric's validations are kept: one health check type and one NEG type per
entry, a Cloud Run or Cloud Functions NEG's target, an internet NEG
endpoint's fqdn or address, at most one port, the protocol, the
session affinities and balancing modes. Name lengths: with a
`namePrefix`, a derived name over 63 characters stops the render; a name
set under `names` is always checked.

## Outputs

| Output | From |
| --- | --- |
| `address` | the forwarding rule's `spec.ipAddress.ip`: the address given, or the one Config Connector writes back; else null |
| `backend_service_generated_ids` | each backend service's `status.generatedId`, by key (null until reported) |
| `backend_service_ids` | `projects/<project>/regions/<region>/backendServices/<name>`, by key |
| `backend_service_names` | each backend service's name, by key |
| `group_ids` | `projects/<project>/zones/<zone>/instanceGroups/<name>`, by key |
| `health_check_ids` | `projects/<project>/regions/<region>/healthChecks/<name>`, by key |
| `id` | `projects/<project>/regions/<region>/forwardingRules/<name>` |
| `neg_ids` | every NEG's ID, by key: `projects/<project>/zones/<zone>/networkEndpointGroups/<name>` or `.../regions/<region>/...` |
| `url_map_id` | `projects/<project>/regions/<region>/urlMaps/<name>` |

Not an output: fabric's whole `forwarding_rule` (`docs/design.md`,
"Outputs").

## Differences from the module

- **Map and object defaults.** `health_check_configs` and
  `urlmap_config` default to null, and the chart applies fabric's
  defaults when they are null (Helm merges maps where Terraform replaces
  them).
- **Secrets.** A created certificate's private key and IAP's client
  secret are Secret references, not values (above).
- **Regional NEGs as backends** (serverless, PSC, internet) are
  referenced by their ID, since a backend's NEG reference by name is to a
  zonal group.
- **Route rules' service** is a URL in `ComputeURLMap`, so a backend
  service key becomes its ID,
  `projects/<project>/regions/<region>/backendServices/<name>`.
- **Instance groups and NEGs together**, **duplicate backends**: as in
  `net-lb-app-int-cross-region`.

## From Terraform

A `module "x" { source = ".../modules/net-lb-app-ext-regional" ... }` block's
arguments become the values unchanged, apart from `context` (refused:
write the values the lookups stand for) and the secrets (give the
reference to where each is kept). Every resource keeps its name, so it
adopts what Terraform made (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/net-lb-app-ext-regional`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/net-lb-app-ext-regional)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=net-lb-app-ext-regional`: render cases from fabric's README,
`basic` ("Minimal Example"), `https`, `negs` (every NEG type, internet
endpoints); and `full` (every variable the chart carries, with
every URL map block), `names` and `mock`. Each is validated against
Config Connector v1.160.0's CRD schemas and config-connector-extensions'
CRDs; the refusals are in `tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`, with the network
from `tests/mock.needs.yaml`). On 2026-11-15 (offload VM `off-catalog-1115102556-2238`,
config-connector-extensions `384d07c0`) all 12 objects became Ready, the
outputs evaluated and all 12 were deleted. The HTTPS proxy
and its certificates, the PSC and internet NEGs are only render-tested.

## Changes

- **0.2.0** (2026-11-15): internet NEG endpoints render config-connector-extensions'
  `v1beta1` `ComputeRegionNetworkEndpoint` and need no `allow_alpha` (now
  a no-op).
- **0.1.0** (2026-11-15): the port.
