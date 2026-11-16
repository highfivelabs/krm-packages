# eks

An Amazon EKS cluster with its IAM role, KMS key, security groups, log
group, access entries, add-ons, identity providers, EKS managed node
groups and Fargate profiles, as AWS Controllers for Kubernetes (ACK)
objects. A port of terraform-aws-modules'
[`terraform-aws-eks`](https://github.com/terraform-aws-modules/terraform-aws-eks/tree/91f11ea01656c26b8e6b8eb1b74f19863f60056c)
at v21.29.0 (`91f11ea0`), Apache-2.0; modified (a translation of the
module's variables into a Helm chart). Bundt Package `aws-eks`
(`package.yaml`). Checked against ACK's eks-controller v1.26.0,
iam-controller v1.11.3, ec2-controller v1.23.1, kms-controller v1.7.0 and
cloudwatchlogs-controller v1.10.0 (`UPSTREAM.md`, "ACK controllers").

## Example

```yaml
name: example
kubernetes_version: "1.33"
endpoint_public_access: true
enable_irsa: false                        # the OIDC provider is a second component
create_primary_security_group_tags: false # no kind owns EKS's own group
attach_encryption_policy: false           # the key policy already grants the role
account_id: "111122223333"
kms_key_administrators: ["arn:aws:iam::111122223333:role/admin"]
vpc_id: vpc-1234556abcdef
subnet_ids: [subnet-abcde012, subnet-bcde012a, subnet-fghi345a]
addons:
  coredns: {}
  kube-proxy: {}
  vpc-cni: {before_compute: true}
eks_managed_node_groups:
  example:
    use_custom_launch_template: false     # EKS's own launch template
    instance_types: [m5.xlarge]
    min_size: 2
    max_size: 10
    desired_size: 2
```

renders the `Cluster` (`hfl.dev/id` `cluster`), its `Role`
(`cluster-role`), KMS `Key` (`kms-key`) and `Alias` (`kms-alias-cluster`),
`LogGroup` (`log-group`), the cluster and node `SecurityGroup`s
(`security-group`, `node-security-group`), an `Addon` per add-on
(`addon-<key>`), and per node group a `Nodegroup` (`node-group-<key>`) and
its `Role` (`iam-role-node-group-<key>`) (`tests/golden/basic.yaml`).

## What it renders

| Module resource | ACK |
| --- | --- |
| `aws_eks_cluster.this` | `Cluster`: `name`, `version`, `roleRef` (the chart's role) or `roleARN`, `accessConfig`, `bootstrapSelfManagedAddons: false`, `deletionProtection`, `logging`, `computeConfig`, `controlPlaneScalingConfig`, `resourcesVPCConfig` (`securityGroupRefs` to the chart's group, or `securityGroupIDs`), `kubernetesNetworkConfig`, the scheduler, controller manager and API server configs, `outpostConfig`, `encryptionConfig` (`keyRef` to the chart's key, or `keyARN`), `remoteNetworkConfig`, `storageConfig`, `upgradePolicy`, `zonalShiftConfig`, `tags` |
| `aws_cloudwatch_log_group.this` | `LogGroup` `/aws/eks/<cluster>/cluster` |
| `module.kms` (terraform-aws-kms v4.0.0) | `Key` with the module's key policy (Default, KeyOwner, KeyAdministration, KeyUsage with the cluster role first, KeyServiceUsage, source and override documents merged by `Sid`) and an `Alias` per alias (`eks/<cluster>` and `kms_key_aliases`, `targetKeyRef`) |
| `aws_iam_role.this` and its attachments | `Role` trusting `eks.amazonaws.com` (and EC2 on Outposts), with the module's managed policies for a standard, EKS Auto Mode or Outposts cluster, `iam_role_additional_policies`, and the chart's own policies, all by ARN |
| `aws_iam_policy.cluster_encryption`, `aws_iam_policy.custom` | `Policy`: the encryption policy (with `encryption_config.provider_key_arn`) and the EKS Auto Mode custom tags policy, attached by their ARNs |
| `aws_iam_role.eks_auto` and its attachments | `Role` for EKS Auto Mode nodes, whose ARN the cluster's `computeConfig.nodeRoleARN` names |
| `aws_iam_policy.cni_ipv6_policy` | `Policy` `AmazonEKS_CNI_IPv6_Policy` |
| `aws_security_group.cluster`, `aws_security_group.node` and their `aws_security_group_rule`s | `SecurityGroup`s with rules inline; the node group's rules from the cluster group by `groupRef`, `self` as a pair without a group |
| `aws_eks_access_entry`, `aws_eks_access_policy_association` | `AccessEntry` (`clusterRef`) with the associations inline as `accessPolicies` |
| `aws_eks_addon` (both) | `Addon` (`clusterRef`): `name`, `addonVersion`, `configurationValues`, `podIdentityAssociations`, `resolveConflicts` (from `resolve_conflicts_on_update`), `serviceAccountRoleARN` |
| `aws_eks_identity_provider_config` | `IdentityProviderConfig` (`clusterRef`) with `oidc` |
| `module.eks_managed_node_group` | `Nodegroup` (`clusterRef`, `nodeRoleRef`) on EKS's own launch template: subnets, scaling, AMI type and release, version, capacity type, disk size, instance types, labels, taints, remote access, update config; and its `Role` (the submodule's policies by ARN, its role policy inline) |
| `module.fargate_profile` | `FargateProfile` (`clusterRef`, `podExecutionRoleRef`) with its `Role` (trust conditioned on the cluster's profiles, as the submodule builds it) |

Every object takes the cluster by reference, so ACK holds it until the
cluster is synced: the module's `time_sleep` and `depends_on` ordering
have no equivalent and need none.

## Values

Every module variable is a value of the same name, type and default. The
chart adds `names` (`cluster`, `clusterRole`, `securityGroup`,
`nodeSecurityGroup`, `nodeRole`, `encryptionPolicy`), `namePrefix` (on
every name the chart gives, and so on the log group and the KMS alias,
which follow the cluster's), `labels` (Kubernetes only), `account_id` and
`partition` (the module reads them from the provider; `account_id` is
needed by the KMS key policy, a Fargate profile's role trust, the IPv6 CNI
policy's ARN and the policies the cluster role attaches by ARN) and
`deletion_policy`; `tags` is the module's own. `name` (or `names.cluster`)
is required, and so are `vpc_id` (with either security group) and
`subnet_ids`.

Four defaults have to be set off for the module's default call to render
(`upstream.yaml` says why each is a gap): `enable_irsa`,
`create_primary_security_group_tags` (only when there are tags),
`attach_encryption_policy` (with `create_kms_key`, the default) and, per
node group, `use_custom_launch_template`. `create_kms_key` needs
`kms_key_administrators` and `account_id`.

Refused (`upstream.yaml`):

- `enable_irsa: true` and `openid_connect_audiences`,
  `include_oidc_root_ca_thumbprint`, `custom_oidc_thumbprints`: the
  `OpenIDConnectProvider` needs the cluster's OIDC issuer, which exists only
  in the `Cluster`'s status, and ACK's kind takes it only as a string. Make
  the provider in a second component from `cluster_oidc_issuer_url`.
- `create_primary_security_group_tags: true` with tags: `aws_ec2_tag` on
  the group EKS makes has no kind.
- `attach_encryption_policy: true` with the chart's key: the policy names
  the key's ARN, which exists only in the `Key`'s status.
- A node group's custom launch template (`use_custom_launch_template: true`,
  the default) and what only it carries (`ami_id`, user data, block
  devices, metadata, network interfaces, placement, the group's own
  security group, EFA, `attach_cluster_primary_security_group`): ACK's
  `LaunchTemplate` takes security group IDs only, with no reference to the
  node group this render makes. Also `node_repair_config`,
  `warm_pool_config`, `force_update_version`,
  `update_config.update_strategy` and `timeouts`: no field on ACK's
  `Nodegroup`.
- `self_managed_node_groups`: user data needs the cluster's endpoint and
  certificate authority from its status, its launch template the node
  group's ID; autoscaling-controller is not pinned either.
- The cluster group's rule from the node group (`ingress_nodes_443`) is
  not rendered, and `security_group_additional_rules.*.source_node_security_group`
  is refused with the chart's node group: the node group's rules already
  reference the cluster group, and ACK resolves a reference only once the
  referenced group is synced, so two groups that reference each other would
  never be made.
- `additional_security_group_ids` with `create_security_group: true`: ACK's
  `Cluster` cannot take group references and IDs together.
- `addons.*.namespace_config`: no field on ACK's `Addon`. Not carried:
  `most_recent` (an unset `addon_version` takes EKS's default version),
  `resolve_conflicts_on_create`, `preserve`, `before_compute`, `timeouts`.
- `force_update_version`, `service_ipv6_cidr` (no field on ACK's
  `Cluster`), `kms_key_deletion_window_in_days`,
  `kms_key_rotation_period_in_days` (on ACK's `Key`),
  `cloudwatch_log_group_class` (on ACK's `LogGroup`).
- `timeouts`, `addons_timeouts`, `dataplane_wait_duration`, each Fargate
  profile's `timeouts`: ACK retries until synced, and Bundt waits.
- The `*_use_name_prefix` variables and `prefix_separator` are accepted,
  and name as given: Terraform's random suffix would change between
  renders.

`enable_cluster_creator_admin_permissions: true` sets the cluster's
`bootstrapClusterCreatorAdminPermissions`: the identity that creates the
cluster, ACK's eks-controller role, gets cluster admin.

## Outputs

`cluster_arn`, `cluster_certificate_authority_data`, `cluster_endpoint`,
`cluster_id`, `cluster_name`, `cluster_oidc_issuer_url`,
`cluster_dualstack_oidc_issuer_url`, `oidc_provider`, `cluster_version`,
`cluster_platform_version`, `cluster_status`,
`cluster_primary_security_group_id`, `cluster_service_cidr`,
`cluster_ip_family`, `cluster_control_plane_scaling_tier`, `kms_key_arn`,
`kms_key_id`, `kms_key_policy`, `cluster_security_group_id` and
`node_security_group_id` (and their ARNs, built as the provider builds
them), `cluster_iam_role_name`, `cluster_iam_role_arn`,
`cluster_iam_role_unique_id`, the EKS Auto Mode `node_iam_role_*`,
`cloudwatch_log_group_name`, `cloudwatch_log_group_arn`, and
`access_entries`, `cluster_addons`, `cluster_identity_providers`,
`eks_managed_node_groups` and `fargate_profiles` (each one's ARN by key),
`eks_managed_node_groups_autoscaling_group_names`. Not provided:
`access_policy_associations` (inline on each access entry),
`oidc_provider_arn` and `cluster_tls_certificate_sha1_fingerprint`
(`enable_irsa` is refused), the self-managed node group outputs (refused).

## Tests

`task test CHART=aws/eks`: golden renders validated against the pinned
controllers' CRDs (a cluster with a KMS key, add-ons and a node group; most
of the chart with a given key; EKS Auto Mode; existing role and groups on
IPv6; Outposts; names in full), and the refusal cases.
`tests/mock.values.yaml` is the case the fake-AWS run applies, in the VPC
and subnets `tests/mock.needs.yaml` makes (`docs/aws.md`, "Charts").
