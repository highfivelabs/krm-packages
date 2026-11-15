# net-lb-app-int

A regional internal Application Load Balancer (an L7 HTTP or HTTPS ILB):
a forwarding rule in a subnet, an HTTP or HTTPS target proxy with its SSL
certificates (existing, or created from a PEM whose private key is a
Kubernetes Secret), a URL map, regional backend services, global or
regional health checks, optional unmanaged instance groups, network
endpoint groups (zonal, hybrid, serverless, internet, Private Service
Connect) and a PSC service attachment, as Config Connector objects. A port
of Cloud Foundation Fabric's
[`modules/net-lb-app-int`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/net-lb-app-int)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-net-lb-app-int` (`package.yaml`). Checked against Config Connector
v1.160.0 (`ComputeForwardingRule`, `ComputeTargetHTTPProxy`,
`ComputeTargetHTTPSProxy`, `ComputeSSLCertificate`, `ComputeURLMap`,
`ComputeBackendService`, `ComputeHealthCheck`, `ComputeInstanceGroup`,
`ComputeNetworkEndpointGroup`, `ComputeRegionNetworkEndpointGroup`,
`ComputeServiceAttachment`) and config-connector-extensions'
`ComputeNetworkEndpoint` (`compute.gcp.hfl.dev/v1beta1`) and
`ComputeRegionNetworkEndpoint` (`v1alpha1`) at `384d07c0`.

Regional internal application load balancers need a proxy-only subnet
with purpose `REGIONAL_MANAGED_PROXY` in the region (net-vpc's
`subnets_proxy_only`); the chart does not make it.

The URL map, backend services, health checks, groups, NEGs and SSL
certificates are rendered by `templates/_netlbapp.tpl`, which
`net-lb-app-ext` and `net-lb-app-ext-regional` carry identical copies of; each chart's
`netlbapp.cfg` says what differs (global or regional, the scheme).

## Example

```yaml
name: ilb-test
project_id: my-project
region: europe-west1
backend_service_configs:
  default:
    backends:
      - group: projects/my-project/zones/europe-west1-b/instanceGroups/my-ig
vpc_config:
  network: projects/my-project/global/networks/my-vpc
  subnetwork: projects/my-project/regions/europe-west1/subnetworks/my-subnet
```

renders the forwarding rule `ilb-test` (HTTP, port 80, premium tier) in
`europe-west1`, the regional target HTTP proxy and URL map `ilb-test`,
the regional backend service `ilb-test-default` and its global HTTP
health check `ilb-test-default` (`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.targetProxy` and `names.urlMap` set those cloud names in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, every name the chart derives or is given in an entry starts with `<namePrefix>-`; empty changes nothing |
| `labels` | fabric's labels: Kubernetes labels on every object; an unprefixed key also becomes a Google Cloud label on the forwarding rule (the only resource here with labels) |
| `tags` | not a fabric variable: cloud labels on the forwarding rule |
| `name` | the forwarding rule's, proxy's, URL map's and service attachment's name, and the base of every derived name |
| `project_id` | every object's project, unless an entry sets its own |
| `region` | the region of the forwarding rule, proxy, URL map, backend services, certificates, service attachment and regional health checks |
| `vpc_config` | `network` and `subnetwork` of the forwarding rule, and of zonal NEGs without their own |
| `address` | the forwarding rule's address: an IP, or an address's ID or self link |
| `global_access` | the forwarding rule's `allowGlobalAccess`, when set |
| `network_tier_premium` | `PREMIUM` (the default) or `STANDARD` |
| `protocol` | `HTTP` (a `ComputeTargetHTTPProxy`) or `HTTPS` (a `ComputeTargetHTTPSProxy`), and the backend services' protocol unless they set one |
| `ports` | the forwarding rule's port (at most one; 443 for HTTPS, 80 for HTTP) |
| `description` | the forwarding rule's, service attachment's and NEGs' description |
| `http_proxy_config`, `https_proxy_config` | the proxy's name and description; for HTTPS also Certificate Manager certificates and the SSL policy |
| `ssl_certificates` | the HTTPS proxy's certificates: `certificate_ids` (existing, by ID) and `create_configs`, one regional `ComputeSSLCertificate` per key named `<name>-<key>` unless it has a name, the certificate (public) as given and the private key a reference to a Kubernetes Secret, `{secretName, key}` (`docs/secrets.md`) |
| `urlmap_config` | the regional `ComputeURLMap`: description, default service, default redirect, default route action, header action, host rules, path matchers (default service, default redirect, default route action, header action, path rules, route rules with their match rules), tests. A service is the key of one of the chart's backend services, else an existing one's ID or self link. `null` (the default) is fabric's default, `{default_service: default}` |
| `backend_service_configs` | one regional `ComputeBackendService` (`INTERNAL_MANAGED`) per key, named `<name>-<key>` unless it has a name: description, cookie TTL, draining timeout, health checks (keys of the chart's or existing IDs; `["default"]` unless set, `[]` for none), locality LB policy, logging, port name, protocol, security policy, session affinity, subsetting (`enable_subsetting`), timeout, circuit breakers, consistent hash, IAP (the client secret a Secret reference), outlier detection and the backends. Refused in part (below) |
| `health_check_configs` | one `ComputeHealthCheck` per key, named `<name>-<key>` unless it has a name, of one type: global, or in `region` with `is_regional: true`. `null` (the default) is fabric's default; `{}` makes none |
| `group_configs` | one unmanaged `ComputeInstanceGroup` per key |
| `neg_configs` | per key, in the entry's project or the chart's, one of: `gce`, a zonal `ComputeNetworkEndpointGroup` (`GCE_VM_IP_PORT`; network and subnetwork the entry's or `vpc_config`'s) with one `ComputeNetworkEndpoint` per endpoint; `hybrid`, the same of type `NON_GCP_PRIVATE_IP_PORT`; `cloudrun`, a serverless `ComputeRegionNetworkEndpointGroup` (service, tag or URL mask); `internet`, a `ComputeRegionNetworkEndpointGroup` (`INTERNET_FQDN_PORT`, or `INTERNET_IP_PORT` with `use_fqdn: false`) on `vpc_config.network` with one `ComputeRegionNetworkEndpoint` per endpoint (alpha: `allow_alpha`); `psc`, a `ComputeRegionNetworkEndpointGroup` to a PSC target service. Named `<name>-<key>` |
| `service_attachment` | one `ComputeServiceAttachment` for the forwarding rule, named `name`: NAT subnets, automatic or manual acceptance, accept and reject lists, proxy protocol. `domain_name` and `reconcile_connections` are refused (below) |
| `service_directory_registration` | refused (below) |
| `context` | refused: Bundt expressions (`${...}`) and `valueFrom` replace fabric's `$context` lookups |
| `allow_alpha` | not a fabric variable: accept config-connector-extensions' `v1alpha1` `ComputeRegionNetworkEndpoint` (internet NEG endpoints) |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps every resource |

Refused, by name:

- `backend_service_configs.<key>.tls_settings`, and
  `log_config.optional_mode` and `optional_fields`:
  `ComputeBackendService` has no `tlsSettings`, and no `optionalMode` or
  `optionalFields` in `logConfig`.
- `neg_configs.<key>.internet.endpoints` without `allow_alpha: true`:
  config-connector-extensions' `ComputeRegionNetworkEndpoint` is
  `v1alpha1`.
- `service_attachment.domain_name` and `reconcile_connections`:
  `ComputeServiceAttachment` has neither field. Any value of
  `reconcile_connections` is refused, `false` too.
- `service_directory_registration`: `ComputeForwardingRule`'s
  `serviceDirectoryRegistrations` takes a service only, not a namespace.
- `ssl_certificates.create_configs.<key>.private_key` and
  `backend_service_configs.<key>.iap_config.oauth2_client_secret` given as
  a value: they take a reference to a Kubernetes Secret,
  `{secretName, key}`, rendered as Config Connector's
  `valueFrom.secretKeyRef` (`docs/secrets.md`).

fabric's validations are kept: one health check type and one NEG type per
entry, a Cloud Run NEG's target, at most one port, the protocol, the
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
| `health_check_ids` | `projects/<project>/global/healthChecks/<name>` (or `regions/<region>`), by key |
| `id` | `projects/<project>/regions/<region>/forwardingRules/<name>` |
| `neg_ids` | `projects/<project>/zones/<zone>/networkEndpointGroups/<name>`, by key |
| `psc_neg_ids`, `regional_neg_ids` | `projects/<project>/regions/<region>/networkEndpointGroups/<name>`, by key, of the PSC and the serverless NEGs |
| `service_attachment_id` | `projects/<project>/regions/<region>/serviceAttachments/<name>`, or null |
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

A `module "x" { source = ".../modules/net-lb-app-int" ... }` block's
arguments become the values unchanged, apart from `context` (refused:
write the values the lookups stand for) and the secrets (give the
reference to where each is kept). Every resource keeps its name, so it
adopts what Terraform made (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/net-lb-app-int`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/net-lb-app-int)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=net-lb-app-int`: render cases from fabric's README,
`basic` ("Minimal Example"), `https`, `certificate-manager`,
`health-check`, `negs` (every NEG type, internet endpoints alpha),
`service-attachment`; and `full` (every variable the chart carries, with
every URL map block), `names` and `mock`. Each is validated against
Config Connector v1.160.0's CRD schemas and config-connector-extensions'
CRDs; the refusals are in `tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`, with the network
and subnet from `tests/mock.needs.yaml`): see "Changes" for the run. The
HTTPS proxy and its certificates, the PSC NEG, the internet NEG's
endpoints (alpha) and the service attachment are only render-tested.

## Changes

- **0.1.0** (2026-11-15): the port.
