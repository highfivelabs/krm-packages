# net-vpn-static

A Classic VPN gateway with its external address and the ESP, UDP 500 and
UDP 4500 forwarding rules that bring IPsec traffic to it, as Config
Connector objects. A port of Cloud Foundation Fabric's
[`modules/net-vpn-static`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/net-vpn-static)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-net-vpn-static` (`package.yaml`). Checked against Config Connector
v1.160.0.

## Example

```yaml
project_id: my-project
region: europe-west1
name: vpn-test
network: projects/my-project/global/networks/my-network
remote_ranges: [10.10.0.0/24]
tunnels:
  remote-0:
    peer_ip: 1.1.1.1
    shared_secret: {secretName: vpn-test, key: remote-0}
    traffic_selectors: {local: [0.0.0.0/0], remote: [0.0.0.0/0]}
```

renders the `ComputeTargetVPNGateway` `vpn-test`, the `ComputeAddress`
`vpn-vpn-test`, the `ComputeForwardingRule`s `vpn-vpn-test-esp`,
`-udp-500` and `-udp-4500`, the `ComputeVPNTunnel` `vpn-test-remote-0`
(its shared secret read from key `remote-0` of the Kubernetes Secret
`vpn-test`) and the `ComputeRoute` `vpn-vpn-test-remote-0-10-10-0-0-24`
(`tests/golden/basic.yaml` has a second tunnel and range).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names.gateway`, `names.address` | not fabric variables: the gateway's and its address's names in full, used as given (no `namePrefix`) and checked (63 characters, a leading letter) (`docs/design.md`, "Names, labels and tags") |
| `namePrefix` | not a fabric variable: an optional convenience; when set, every name the chart derives starts with `<namePrefix>-`; empty changes nothing |
| `labels`, `tags` | Kubernetes labels on every object; unprefixed ones and `tags` reach the resources that take Google Cloud labels (the address, the forwarding rules) |
| `name`, `project_id`, `region`, `network` | the gateway (`name`), the address (`vpn-<name>`) and forwarding rules (`vpn-<name>-esp`, `-udp-500`, `-udp-4500`), their project and region; the gateway's `networkRef` |
| `gateway_address_create`, `gateway_address` | the `ComputeAddress` the forwarding rules use (true, the default), or the given address (false) |
| `remote_ranges`, `route_priority` | a `ComputeRoute` per tunnel and range, `vpn-<name>-<tunnel>-<the range's numbers joined by ->` as fabric names it, its next hop the tunnel, with `route_priority` |
| `tunnels` | a `ComputeVPNTunnel` per tunnel, `<name>-<key>`, on the gateway, to `peer_ip`, with `traffic_selectors` (`localTrafficSelector`, `remoteTrafficSelector`) and `ike_version` (2 by default); `shared_secret`, required, a reference to a Kubernetes Secret, `{secretName, key}` ([secrets](../../../docs/secrets.md)), as `sharedSecret.valueFrom.secretKeyRef`: a string is refused, and so is none (fabric would generate one; the catalog never generates a secret). `cipher_suite` is refused: `ComputeVPNTunnel` has no `cipherSuite` field (a gap) |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the cloud resources |

## Outputs

`address` (the created address, else `gateway_address`), `id`, `name` and
`self_link` of the gateway, `tunnel_names` and `tunnel_self_links` (keyed
by tunnel). Not outputs: `random_secret` (the chart makes no secret), and
fabric's whole objects (`gateway`, `tunnels`; `docs/design.md`,
"Outputs").

## From Terraform

A `module "x" { source = ".../modules/net-vpn-static" ... }` block's
arguments become the values unchanged, but a tunnel's `shared_secret`,
which becomes a reference to the Kubernetes Secret that holds it
(`examples/net-vpn-static/`). To adopt an existing gateway, keep its names:
Config Connector acquires the resources of those names (`docs/design.md`,
"From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/net-vpn-static`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/net-vpn-static)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml) records
the repository, path, release, commit, licence and the date last synced, and
the module is listed in [`upstream/modules.yaml`](../../../upstream/modules.yaml)
with every other module the catalog tracks. To take a newer release, see
[`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=net-vpn-static`: render cases `basic` (two tunnels, two
remote ranges), `address` and `names`, each validated against Config Connector v1.160.0's CRD schemas,
and the refusals in `tests/*.fail.yaml`.

Fake cloud (`task test-mock MOCK_CHARTS=net-vpn-static`, 2026-11-12): the
gateway, its address and its three forwarding rules on a network from
`tests/mock.needs.yaml`: every object Ready, the outputs evaluated, every
object deleted.

Applied live on 2026-11-12 (`hack/live-apply.sh` with
`LIVE_CHARTS=net-vpn-static`, offload VM as `catalog-live-tests`): the
gateway, its address and its three forwarding rules on a network made for
the run: all Ready, the outputs evaluated (`address` the allocated IP),
deleted, and gone from Google Cloud.

## Changes

- **0.2.0** (2026-11-15): `tunnels` and their routes; a tunnel's shared secret is a reference to a Kubernetes Secret. `tunnel_names` and `tunnel_self_links` outputs. mockgcp has no VPN tunnels: render-tested only.
- **0.1.0** (2026-11-12): the port.
