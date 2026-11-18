# opensearch

An Amazon OpenSearch Service domain with its access policy, SAML options,
VPC endpoints, CloudWatch log groups and security group, as AWS Controllers
for Kubernetes (ACK) objects. A port of terraform-aws-modules'
[`terraform-aws-opensearch`](https://github.com/terraform-aws-modules/terraform-aws-opensearch/tree/370988b1bd771fd26037e8b1471ddd0ae651c602)
at v2.11.1 (`370988b1`), Apache-2.0; modified (a translation of the module's
variables and logic into a Helm chart). Bundt Package `aws-opensearch`
(`package.yaml`). Checked against ACK's opensearchservice-controller v1.10.0
(`UPSTREAM.md`, "ACK controllers"), ec2-controller v1.23.1 and
cloudwatchlogs-controller v1.10.0. The module's `modules/collection`
submodule (OpenSearch Serverless) is not part of this chart.

## What it renders

| Module resource | When | Object (`hfl.dev/id`) |
| --- | --- | --- |
| `aws_opensearch_domain` | `create` | `opensearchservice` `Domain` (`domain`) |
| `aws_opensearch_domain_policy` | `enable_access_policy`, with `access_policy_statements` or `access_policies` | the Domain's `spec.accessPolicies` |
| `aws_opensearch_domain_saml_options` | `create_saml_options` | the Domain's `spec.advancedSecurityOptions.sAMLOptions` |
| `aws_opensearch_vpc_endpoint` | `vpc_endpoints` | `opensearchservice` `VPCEndpoint` (`vpc-endpoint-<key>`, `domainRef`) |
| `aws_cloudwatch_log_group` | `create_cloudwatch_log_groups`, one for each enabled `log_publishing_options` entry | `cloudwatchlogs` `LogGroup` (`log-group-<type>`) |
| `aws_security_group`, `aws_vpc_security_group_*_rule` | `create_security_group` and `vpc_options` | `ec2` `SecurityGroup` (`security-group`, rules inline; the Domain takes it by `securityGroupRefs`) |

## Differences from Terraform

- Two values the module reads from the provider or the account are the
  chart's own: `account_id` (with `region`) writes the domain's log group
  ARNs and the domain's ARN in a policy statement without `resources`, and
  `vpc_id` is the security group's VPC (the module reads it from the first
  subnet).
- Not carried, each refused by name (`upstream.yaml`): package associations
  and outbound connections (no kind in opensearchservice-controller),
  `deployment_strategy_options`, `identity_center_options`, `timeouts`, the
  log group class, `jwt_options.jwks_url`, `aiml_options.s3_vectors_engine`
  and `.serverless_vector_acceleration`, `cluster_config.node_options`, an
  `auto_tune_options.rollback_on_disable` other than NO_ROLLBACK, and the
  source and override policy documents (write the statements in
  `access_policy_statements`).
- **`create_cloudwatch_log_resource_policy` (true by default) is refused**
  while the chart makes log groups: no kind makes a CloudWatch Logs resource
  policy, in ACK or in ack-extensions, and the domain cannot publish to its
  log groups without one. Set it to false and give the account a resource
  policy that lets `es.amazonaws.com` write to them.
- A password is never in values: `master_user_options.master_user_password`
  is refused; `master_user_options.master_user_password_secret_ref` (`name`,
  `key`, `namespace`) takes a Kubernetes Secret, which ACK reads.
- The domain's log groups are not referenced from the Domain (ACK takes the
  ARN as a string), so the Domain may fail its first reconciles until they
  exist; ACK retries.
- `security_group_use_name_prefix: true` (the default) names the group as
  given, without Terraform's random suffix; `vpc_options.security_group_ids`
  with the chart's own group is refused (ACK takes group references or IDs,
  not both); per-rule tags are not carried.
