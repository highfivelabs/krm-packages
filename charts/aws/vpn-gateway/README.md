# vpn-gateway

A Site-to-Site VPN connection between a customer gateway and a virtual
private gateway or a transit gateway, its static routes, and the route
propagation of the gateway to route tables, as ack-extensions' `VPNConnection`
and `VPNGatewayRoutePropagation` objects. A port of terraform-aws-modules'
[`terraform-aws-vpn-gateway`](https://github.com/terraform-aws-modules/terraform-aws-vpn-gateway/tree/b4a481d06144addf271996c10ebdb688b06362f6)
at v4.0.0 (`b4a481d0`), Apache-2.0; modified (a translation of the module's
variables and logic into a Helm chart). Bundt Package `aws-vpn-gateway`
(`package.yaml`). ACK's ec2-controller (v1.23.1) has no VPN connection kind,
so every object here is an ack-extensions kind and renders only with
`allow_alpha: true` (`docs/design.md`, "Alpha kinds").

## What it renders

| Module resource | When | Object (`hfl.dev/id`) |
| --- | --- | --- |
| `aws_vpn_connection` (all four variants) | `create_vpn_connection` | `ec2.aws.hfl.dev` `VPNConnection` (`vpn-connection`) |
| `aws_vpn_connection_route` | static routes only, not a transit gateway | `VPNConnection` `spec.staticRoutes` |
| `aws_vpn_gateway_route_propagation` | not a transit gateway | `VPNGatewayRoutePropagation` (`route-propagation-<i>`) |

The module's four `aws_vpn_connection` resources differ only in whether
both tunnels' inside CIDRs and both pre-shared keys are given; the chart
sends each only when both tunnels have it, as the module does.

## Differences from Terraform

- `tunnel1_preshared_key` and `tunnel2_preshared_key` take a reference to a
  Kubernetes Secret, `{secretName, key}`, never the key itself
  (`docs/secrets.md`).
- `create_vpn_gateway_attachment` (default true) is refused with `vpc_id`
  on a virtual private gateway: ack-extensions' `VPNGateway` creates and
  attaches a gateway in one object and cannot attach an existing one. Set it
  to false and attach the gateway where it is created.
- With `connect_to_transit_gateway`, the tags the module puts on the
  transit gateway attachment (`aws_ec2_tag`) are not applied: `VPNConnection`
  reports no attachment ID for a `ResourceTag` to name.
- The tunnels' inside addresses and the customer gateway configuration are
  not Package outputs: `VPNConnection`'s status does not carry them.
