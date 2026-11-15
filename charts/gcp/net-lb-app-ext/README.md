# net-lb-app-ext

A global external Application Load Balancer, classic (`EXTERNAL`) or
global (`EXTERNAL_MANAGED`): global forwarding rules, an HTTP or HTTPS
target proxy with its SSL certificates (existing, created from a PEM whose
private key is a Kubernetes Secret, or Google-managed), a URL map,
backend services and backend buckets, health checks, optional unmanaged
instance groups and network endpoint groups (zonal, hybrid, serverless,
internet, Private Service Connect), as Config Connector objects. A port of
Cloud Foundation Fabric's
[`modules/net-lb-app-ext`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/net-lb-app-ext)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-net-lb-app-ext` (`package.yaml`). Checked against Config Connector
v1.160.0 (`ComputeForwardingRule`, `ComputeTargetHTTPProxy`,
`ComputeTargetHTTPSProxy`, `ComputeSSLCertificate`,
`ComputeManagedSSLCertificate`, `ComputeURLMap`, `ComputeBackendService`,
`ComputeBackendBucket`, `ComputeHealthCheck`, `ComputeInstanceGroup`,
`ComputeNetworkEndpointGroup`, `ComputeRegionNetworkEndpointGroup`,
`ComputeGlobalNetworkEndpoint` (`v1alpha1`)) and config-connector-extensions'
`ComputeNetworkEndpoint` and `ComputeGlobalNetworkEndpointGroup`
(`compute.gcp.hfl.dev/v1beta1`) at `384d07c0`.

The URL map, backend services, health checks, groups, NEGs and SSL
certificates are rendered by `templates/_netlbapp.tpl`, which
`net-lb-app-int` and `net-lb-app-ext-regional` carry identical copies of; each chart's
`netlbapp.cfg` says what differs (global or regional, the scheme).

## Example

```yaml
name: glb-test-0
project_id: my-project
protocol: HTTPS
backend_service_configs:
  default:
    backends:
      - group: projects/my-project/zones/europe-west8-b/instanceGroups/my-ig-b
    protocol: HTTP
ssl_certificates:
  managed_configs:
    managed:
      domains: [glb-test.example.org]
```

renders the forwarding rule `glb-test-0` (HTTPS, port 443, `EXTERNAL`),
the target HTTPS proxy and URL map `glb-test-0`, the managed certificate
`glb-test-0-managed`, the backend service `glb-test-0-default` and its
HTTP health check `glb-test-0-default` (`tests/golden/https.yaml`, which
also has an existing and a created certificate).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.targetProxy` and `names.urlMap` set those cloud names in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, every name the chart derives or is given in an entry starts with `<namePrefix>-`; empty changes nothing |
| `labels` | fabric's labels: Kubernetes labels on every object; an unprefixed key also becomes a Google Cloud label on the forwarding rules (the only resources here with labels) |
| `tags` | not a fabric variable: cloud labels on the forwarding rules |
| `name` | the base of every derived name, and the URL map's and proxy's name |
| `project_id` | every object's project, unless a backend service, bucket, health check, group or serverless NEG sets its own |
| `forwarding_rules_config` | one global `ComputeForwardingRule` (TCP) per key, named `name` for the key `""` and `<name>-<key>` otherwise unless it has a name: address (an IP, or an address's ID or self link), IPv6 (`ipVersion`, only without an address, as fabric), description, port (at most one; 443 for HTTPS, 80 for HTTP). `null` (the default) is fabric's default, `{"": {}}` |
| `use_classic_version` | `true` (the default): `EXTERNAL` forwarding rules and backend services; `false`: `EXTERNAL_MANAGED`, and the proxies take `http_keepalive_timeout` |
| `protocol` | `HTTP` (a `ComputeTargetHTTPProxy`) or `HTTPS` (a `ComputeTargetHTTPSProxy`), and the backend services' protocol unless they set one |
| `description` | the forwarding rules' and NEGs' description (a NEG's own first) |
| `http_proxy_config`, `https_proxy_config` | the proxy's name, description and keep-alive timeout (non-classic only, as fabric); for HTTPS also the certificate map, Certificate Manager certificates, QUIC override, SSL policy and mTLS policy (`serverTlsPolicyRef`) |
| `ssl_certificates` | the HTTPS proxy's certificates, in fabric's order: `certificate_ids` (existing, by ID); `create_configs`, one `ComputeSSLCertificate` per key named `<name>-<key>`, the certificate (public) as given and the private key a reference to a Kubernetes Secret, `{secretName, key}` (`docs/secrets.md`); `managed_configs`, one `ComputeManagedSSLCertificate` per key, named `<name>-<key>` unless it has a name, with its domains and description |
| `urlmap_config` | the `ComputeURLMap`: description, default service, default custom error response policy, default redirect, default route action, header action, host rules, path matchers (default service, default custom error response policy, default redirect, default route action, header action, path rules, route rules with their match rules), tests. A service is the key of one of the chart's backend services or buckets, else an existing one's ID or self link (a bucket when it holds `/backendBuckets/`). `null` (the default) is fabric's default, `{default_service: default}`. A path or route rule's `custom_error_response_policy` is refused (below) |
| `backend_buckets_config` | one `ComputeBackendBucket` per key, named `<name>-<key>` unless it has a name: the bucket, compression mode, custom response headers, description, edge security policy, CDN and its policy |
| `backend_service_configs` | one global `ComputeBackendService` per key, named `<name>-<key>` unless it has a name: description, cookie TTL, compression mode, draining timeout, custom request and response headers, CDN and its policy, health checks (keys of the chart's or existing IDs; `["default"]` unless set, `[]` for none), locality LB policy or policies, logging, port name (`lower(protocol)` unless set, as fabric), protocol, security policy, session affinity, timeout, circuit breakers, consistent hash, IAP (the client secret a Secret reference), outlier detection, security settings and the backends (group, balancing mode, capacity scaler, description, `max_connections` and `max_rate` per group, endpoint or instance, max utilization). Refused in part (below) |
| `health_check_configs` | one global `ComputeHealthCheck` per key, named `<name>-<key>` unless it has a name, of one type (gRPC, HTTP, HTTP/2, HTTPS, TCP, SSL). `null` (the default) is fabric's default; `{}` makes none |
| `group_configs` | one unmanaged `ComputeInstanceGroup` per key |
| `neg_configs` | per key, one of: `gce`, a zonal `ComputeNetworkEndpointGroup` (`GCE_VM_IP_PORT`) with one `ComputeNetworkEndpoint` per endpoint; `hybrid`, the same of type `NON_GCP_PRIVATE_IP_PORT`; `cloudrun` or `cloudfunction`, a serverless `ComputeRegionNetworkEndpointGroup` in the entry's project; `internet`, a `ComputeGlobalNetworkEndpointGroup` (`INTERNET_FQDN_PORT`, or `INTERNET_IP_PORT` with `use_fqdn: false`) with one `ComputeGlobalNetworkEndpoint` per endpoint (alpha: `allow_alpha`); `psc`, a `ComputeRegionNetworkEndpointGroup` to a PSC target service. Named `<name>-<key>`. `serverless_deployment` is refused (below) |
| `context` | refused: Bundt expressions (`${...}`) and `valueFrom` replace fabric's `$context` lookups |
| `allow_alpha` | not a fabric variable: accept Config Connector's `v1alpha1` `ComputeGlobalNetworkEndpoint` (internet NEG endpoints) |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps every resource |

Refused, by name:

- `backend_service_configs.<key>.tls_settings`, `backends[].preferred`
  (`true`), `security_settings.aws_v4_authentication`, and
  `log_config.optional_mode` and `optional_fields`: `ComputeBackendService`
  has no `tlsSettings`, no backend `preference`, no `awsV4Authentication`
  in `securitySettings`, and no `optionalMode` or `optionalFields` in
  `logConfig`.
- `neg_configs.<key>.serverless_deployment`:
  `ComputeRegionNetworkEndpointGroup` has no `serverlessDeployment`.
- `neg_configs.<key>.internet.endpoints` without `allow_alpha: true`:
  Config Connector's `ComputeGlobalNetworkEndpoint` is `v1alpha1`.
- `urlmap_config.path_matchers.<key>.path_rules[].custom_error_response_policy`
  and `route_rules[].custom_error_response_policy`: `ComputeURLMap`'s
  `pathRule` and `routeRules` have no `customErrorResponsePolicy`.
- `ssl_certificates.create_configs.<key>.private_key` and
  `backend_service_configs.<key>.iap_config.oauth2_client_secret` given as
  a value: they take a reference to a Kubernetes Secret,
  `{secretName, key}`, rendered as Config Connector's
  `valueFrom.secretKeyRef`, so no secret is in the values or the manifest
  (`docs/secrets.md`).

fabric's validations are kept: one health check type and one NEG type per
entry, a Cloud Run or Cloud Functions NEG's target, at most one port per
forwarding rule, the protocol, the locality LB policies, session
affinities and balancing modes. Name lengths: with a `namePrefix`, a
derived name over 63 characters stops the render; a name set under
`names` is always checked.

## Outputs

| Output | From |
| --- | --- |
| `address` | each forwarding rule's `spec.ipAddress.ip`, by key: the address given, or the one Config Connector writes back; else null (`ComputeForwardingRule`'s status has no address) |
| `backend_service_generated_ids` | each backend service's `status.generatedId`, by key (null until reported) |
| `backend_service_ids` | `projects/<project>/global/backendServices/<name>`, by key |
| `backend_service_names` | each backend service's name, by key |
| `global_neg_ids` | `projects/<project>/global/networkEndpointGroups/<name>`, by key, of the internet NEGs |
| `group_ids` | `projects/<project>/zones/<zone>/instanceGroups/<name>`, by key |
| `health_check_ids` | `projects/<project>/global/healthChecks/<name>`, by key |
| `ids` | `projects/<project>/global/forwardingRules/<name>`, by key |
| `neg_ids` | `projects/<project>/zones/<zone>/networkEndpointGroups/<name>`, by key |
| `psc_neg_ids`, `serverless_neg_ids` | `projects/<project>/regions/<region>/networkEndpointGroups/<name>`, by key |
| `url_map_id` | `projects/<project>/global/urlMaps/<name>` |

Not an output: fabric's whole `forwarding_rules` (`docs/design.md`,
"Outputs").

## Differences from the module

- **Map and object defaults.** Helm merges a map or object given in the
  values into the chart's default, where Terraform replaces it, so
  `forwarding_rules_config`, `health_check_configs` and `urlmap_config`
  default to null, and the chart applies fabric's defaults when they are
  null.
- **Secrets.** A created certificate's private key and IAP's client
  secret are Secret references, not values (above); the certificate
  itself is public and sent as given (trimmed, as fabric).
- **Managed certificates on the proxy** are referenced by ID
  (`projects/<project>/global/sslCertificates/<name>`): a reference by
  name is to a `ComputeSSLCertificate`.
- **Regional and global NEGs as backends** (serverless, PSC, internet) are
  referenced by their ID, since a backend's NEG reference by name is to a
  zonal group; Config Connector retries the backend service until the
  group exists.
- **Route rules' service** and a custom error policy's error service that
  is a backend service are URLs in `ComputeURLMap`, so a key becomes its
  ID.
- **CORS max age, keep-alive timeout and header range values** are
  strings in fabric and integers in Config Connector; the chart converts
  them.
- **Instance groups and NEGs together.** The API refuses a backend
  service whose backends mix instance groups and network endpoint groups;
  fabric does not check it, and neither does the chart.
- **Duplicate backends.** fabric keys backends by group, and Terraform
  stops on a duplicate; the chart keeps the first.

## From Terraform

A `module "x" { source = ".../modules/net-lb-app-ext" ... }` block's
arguments become the values unchanged, apart from `context` (refused:
write the values the lookups stand for) and the secrets (give the
reference to where each is kept). Every resource keeps its name, so it
adopts what Terraform made (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/net-lb-app-ext`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/net-lb-app-ext)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=net-lb-app-ext`: render cases from fabric's README,
`basic` ("Minimal example"), `https`, `certificate-manager`, `buckets`,
`serverless-neg`, `internet-neg` (with endpoints, alpha),
`internet-neg-groups`, `zonal-neg`, `psc-neg`, `forwarding-rules`; and
`full` (every variable the chart carries, with every URL map block),
`names` and `mock`. Each is validated against Config Connector v1.160.0's
CRD schemas and config-connector-extensions' CRDs; the refusals are in
`tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`, with the network
and the bucket from `tests/mock.needs.yaml`). On 2026-11-15 (offload VM `off-catalog-1115102556-2238`,
config-connector-extensions `384d07c0`) all 15 objects became Ready, the
outputs evaluated and all 15 were deleted.
The HTTPS proxy and its certificates, the PSC NEG and the internet NEG's
endpoints (alpha) are only render-tested.

## Changes

- **0.1.0** (2026-11-15): the port.
