# route53

A Route 53 hosted zone, public or private, and its records, as AWS
Controllers for Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-route53`](https://github.com/terraform-aws-modules/terraform-aws-route53/tree/883f987d6bb328c09bd4dfc5a04024534b371d59)
at v6.5.1 (`883f987d`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-route53`
(`package.yaml`). Checked against ACK's route53-controller v1.8.0
(`UPSTREAM.md`, "ACK controllers").

## Example

```yaml
name: example.com
records:
  www: {type: A, ttl: 300, records: [192.0.2.10]}
  blue: {name: app, type: CNAME, ttl: 60, records: [blue.example.net.], set_identifier: blue, weighted_routing_policy: {weight: 90}}
```

renders one `route53.services.k8s.aws/v1alpha1` `HostedZone` (`hfl.dev/id`
`zone`) and a `RecordSet` per record (`record-<key>`), each in the zone by
`hostedZoneRef` (`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_route53_zone.this` | `HostedZone`: `name`, `hostedZoneConfig.comment` (Terraform's default, "Managed by Terraform", when unset), `privateZone` and `vpcs` (a private zone's `vpc`), `delegationSetID` (a public zone's), `tags` |
| `aws_route53_record.this` | `RecordSet` per entry of `records`: `name`, `recordType`, `ttl`, `resourceRecords`, `aliasTarget`, `setIdentifier`, `healthCheckID`, `weight`, `failover`, `geoLocation`, `region` (latency), `multiValueAnswer`, `cidrRoutingConfig` |

A record's name is its `full_name`, else `<name>.<zone>`, else
`<key>.<zone>`, as in the module: ACK appends the zone's domain to a name
without a trailing dot, so the chart gives `full_name` fully qualified.

ACK keeps a private zone's VPC associations equal to `spec.vpcs`, so
associations made elsewhere are removed (`ignore_vpc` is refused). Route 53
needs each VPC's region; the module defaults it to the provider's region,
which a render cannot know, so `vpc_region` is required.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `labels` (Kubernetes only) and `deletion_policy` (`ABANDON`
renders `services.k8s.aws/deletion-policy: retain`); `tags` is the
module's own. The chart names nothing (a zone's name is its domain), so it
takes no `names` keys and `namePrefix` changes nothing. `name` is required.

Refused (`upstream.yaml`):

- `create_zone: false`, `private_zone`, `vpc_id`: the lookup of an
  existing zone; a render cannot look anything up. The zone is another
  component's, and its records belong there.
- `enable_dnssec` and the `dnssec_*` variables: ack-extensions'
  `KeySigningKey` and `DNSSEC` (Cloud Control) take the zone's ID only as a
  string, with no reference to a `HostedZone`, so they cannot be wired to
  the zone this render makes.
- `vpc_association_authorizations`: no ACK or ack-extensions kind.
- `enable_accelerated_recovery`, `force_destroy`: no field on ACK's
  `HostedZone`.
- `ignore_vpc`: see above.
- In `records`: `geoproximity_routing_policy` (no field on ACK's
  `RecordSet`), `allow_overwrite: true` (no UPSERT on create), an alias's
  `zone_id: self` (`aliasTarget` takes a zone only by ID); `timeouts` is
  not carried.
- `timeouts`: ACK retries until synced, and Bundt waits.

## Outputs

`id` (the zone's ID without `/hostedzone/`), `arn` (built as the provider
builds it), `name_servers`, `name`, and `records` (each record's fully
qualified name by key). Not provided: `primary_name_server` (not in
`GetHostedZone`) and the DNSSEC outputs.

## Tests

`task test CHART=aws/route53`: golden renders validated against
route53-controller v1.8.0's CRDs, and the refusal cases.
`tests/mock.values.yaml` is the case the fake-AWS run applies
(`docs/aws.md`, "Charts").
