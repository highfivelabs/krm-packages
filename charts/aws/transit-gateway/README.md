# transit-gateway

An EC2 transit gateway with its VPC attachments, as AWS Controllers for
Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-transit-gateway`](https://github.com/terraform-aws-modules/terraform-aws-transit-gateway/tree/7973ea0890176868b62bb60a694768cec19daf9a)
at v3.3.1 (`7973ea08`), Apache-2.0; modified (a translation of the module's
variables and logic into a Helm chart). Bundt Package `aws-transit-gateway`
(`package.yaml`). Checked against ACK's ec2-controller v1.23.1.

## What it renders

| Module resource | When | Object (`hfl.dev/id`) |
| --- | --- | --- |
| `aws_ec2_transit_gateway` | `create_tgw` | `ec2` `TransitGateway` (`transit-gateway`) |
| `aws_ec2_transit_gateway_vpc_attachment` | one per `vpc_attachments` entry | `ec2` `TransitGatewayVPCAttachment` (`attachment-<key>`), `transitGatewayRef` to the gateway, or `transitGatewayID` from the entry's `tgw_id` when `create_tgw` is false |

## Differences from Terraform

The gateway's own routing and its sharing cannot be wired in one render:
ack-extensions' Cloud Control kinds for the route table, routes,
associations and propagations, and ACK's `ResourceShare` resource list, take
the gateway's or an attachment's ID or ARN as a plain string, known only
after apply, and have no reference field. So:

- `create_tgw_routes` (default true) is refused when it would create
  anything (set it to `false`), and so are an attachment's `tgw_routes`,
  `vpc_route_table_ids`, `tgw_destination_cidr` and
  `transit_gateway_route_table_id`, `transit_gateway_route_table_id` itself,
  `tgw_route_table_tags` and `tgw_default_route_table_tags`.
- `share_tgw` (default true) is refused when it would share the gateway (set
  it to `false`), as are the `ram_*` variables; the share accepter
  (`aws_ram_resource_share_accepter`) has no kind.
- `enable_sg_referencing_support` (default true) is refused unless `false`
  (ACK's `TransitGateway` has no such option; false is not sent), and
  `enable_encryption_support` unless false.
- An attachment's `transit_gateway_default_route_table_association` and
  `_propagation` set to `false` are refused (no option on ACK's
  `TransitGatewayVPCAttachment`).
- `timeouts` is refused (by design: the controllers retry).
