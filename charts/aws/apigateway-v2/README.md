# apigateway-v2

Port of [terraform-aws-modules/terraform-aws-apigateway-v2](https://github.com/terraform-aws-modules/terraform-aws-apigateway-v2/tree/v6.1.1) (`v6.1.1`) to AWS Controllers for Kubernetes (ACK) objects.

## What it renders

- `apigatewayv2.services.k8s.aws/v1alpha1` `API`, `Authorizer`, `VPCLink`, `Integration`, `Route`, `Stage`, `DomainName`, `APIMapping`
- `cloudwatchlogs.services.k8s.aws/v1alpha1` `LogGroup` (when `stage_access_log_settings` is configured with `create_log_group: true`)

## Differences from Terraform

- `ip_address_type` is refused (`API` in `apigatewayv2-controller v1.9.0` has no `ipAddressType` field).
- `route_response.create` and `integration.response` are refused (no ACK `RouteResponse` or `IntegrationResponse` kinds).
- Route 53 alias records (`hosted_zone_name`, `private_zone`, `subdomains`, `subdomain_record_types`) are refused because `RecordSet` has no reference to `DomainName.status.domainNameConfigurations[0].targetDomainName`.
- Pass `domain_name_certificate_arn` instead of creating an ACM certificate inside this chart.
