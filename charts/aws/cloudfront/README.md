# cloudfront

Port of [terraform-aws-modules/terraform-aws-cloudfront](https://github.com/terraform-aws-modules/terraform-aws-cloudfront/tree/v6.7.1) (`v6.7.1`) to AWS Controllers for Kubernetes (ACK) objects.

## What it renders

- `cloudfront.services.k8s.aws/v1alpha1` `OriginAccessControl`, `VPCOrigin`, `CachePolicy`, `OriginRequestPolicy`, `ResponseHeadersPolicy`, `Function`, `Distribution`

## Differences from Terraform

- `create_monitoring_subscription` and `create_connection_function` are refused (no `MonitoringSubscription` or `ConnectionFunction` kinds in `cloudfront-controller v1.11.0`).
- `enable_v2_logging` / `v2_logging` are refused (`DeliverySource` requires the distribution's ARN, which is only in `Distribution.status`).
- `*_key` cross-references inside `origin` / `default_cache_behavior` / `ordered_cache_behavior` are refused because ACK `Distribution` has no `*Ref` fields; pass `*_id` or `function_arn`.
- `retain_on_delete` is refused; use `deletion_policy: ABANDON`.
