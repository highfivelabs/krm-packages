# ack-extensions CRDs

The CRDs of ack-extensions kinds the AWS charts render, copied from that
repository so the render tests (`hack/test.sh`) validate them with
`kubeconform -strict`, as they validate ACK's own kinds against the CRDs
of the controllers pinned in `UPSTREAM.md` ("ACK controllers").
ack-extensions' kinds are outside ACK's releases, so a chart renders them
only with `allow_alpha: true` (`docs/design.md`, "Alpha kinds";
`lib/_ack.tpl`, `ack.alpha`).

| File | Kind | Copied from |
| --- | --- | --- |
| `ec2.aws.hfl.dev_customergateways.yaml` | `CustomerGateway` (`ec2.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `52556a276e902db1c835e2f7cd3b3d034f0465e8` (2026-11-12) |
| `ec2.aws.hfl.dev_launchtemplates.yaml` | `LaunchTemplate` (`ec2.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `7117ca5a` (2026-11-17) |
| `kms.aws.hfl.dev_replicakeys.yaml` | `ReplicaKey` (`kms.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `52556a276e902db1c835e2f7cd3b3d034f0465e8` (2026-11-12) |
| `ec2.cloudcontrol.aws.hfl.dev_vpcblockpublicaccessoptionses.yaml` | `VPCBlockPublicAccessOptions` (`ec2.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `74d322f6` (2026-11-12) |
| `rds.aws.hfl.dev_optiongroups.yaml` | `OptionGroup` (`rds.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `52556a276e902db1c835e2f7cd3b3d034f0465e8` (2026-11-12) |
| `rds.aws.hfl.dev_dbroleassociations.yaml` | `DBRoleAssociation` (`rds.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `52556a276e902db1c835e2f7cd3b3d034f0465e8` (2026-11-12) |
| `ec2.cloudcontrol.aws.hfl.dev_vpcblockpublicaccessexclusions.yaml` | `VPCBlockPublicAccessExclusion` (`ec2.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-12) |
| `ec2.cloudcontrol.aws.hfl.dev_securitygroupvpcassociations.yaml` | `SecurityGroupVpcAssociation` (`ec2.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-12) |
| `ec2.cloudcontrol.aws.hfl.dev_eipassociations.yaml` | `EIPAssociation` (`ec2.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-12) |
| `secretsmanager.cloudcontrol.aws.hfl.dev_resourcepolicies.yaml` | `ResourcePolicy` (`secretsmanager.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-15) |
| `ecr.cloudcontrol.aws.hfl.dev_registrypolicies.yaml` | `RegistryPolicy` (`ecr.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-15) |
| `ecr.cloudcontrol.aws.hfl.dev_registryscanningconfigurations.yaml` | `RegistryScanningConfiguration` (`ecr.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-15) |
| `ecr.aws.hfl.dev_replicationconfigurations.yaml` | `ReplicationConfiguration` (`ecr.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-15) |
| `autoscaling.aws.hfl.dev_autoscalinggroups.yaml` | `AutoScalingGroup` (`autoscaling.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `7117ca5a` (2026-11-17) |
| `autoscaling.aws.hfl.dev_scalingpolicies.yaml` | `ScalingPolicy` (`autoscaling.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-16) |
| `autoscaling.aws.hfl.dev_scheduledactions.yaml` | `ScheduledAction` (`autoscaling.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-16) |
| `autoscaling.cloudcontrol.aws.hfl.dev_warmpools.yaml` | `WarmPool` (`autoscaling.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-16) |
| `eventbridge.aws.hfl.dev_connections.yaml` | `Connection` (`eventbridge.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-16) |
| `eventbridge.aws.hfl.dev_apidestinations.yaml` | `APIDestination` (`eventbridge.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-16) |
| `scheduler.aws.hfl.dev_schedulegroups.yaml` | `ScheduleGroup` (`scheduler.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-16) |
| `scheduler.aws.hfl.dev_schedules.yaml` | `Schedule` (`scheduler.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `2cdca6ca` (2026-11-16) |
| `eventschemas.cloudcontrol.aws.hfl.dev_discoverers.yaml` | `Discoverer` (`eventschemas.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `2cdca6ca` (2026-11-16) |
| `wafv2.aws.hfl.dev_webaclassociations.yaml` | `WebACLAssociation` (`wafv2.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `7117ca5a` (2026-11-17) |
| `lambda.aws.hfl.dev_permissions.yaml` | `Permission` (`lambda.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `7117ca5a` (2026-11-17) |
| `applicationautoscaling.aws.hfl.dev_scheduledactions.yaml` | `ScheduledAction` (`applicationautoscaling.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `7117ca5a` (2026-11-17) |
| `route53.cloudcontrol.aws.hfl.dev_keysigningkeys.yaml` | `KeySigningKey` (`route53.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `7117ca5a` (2026-11-17) |
| `route53.cloudcontrol.aws.hfl.dev_dnssecs.yaml` | `DNSSEC` (`route53.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `7117ca5a` (2026-11-17) |
| `ecs.cloudcontrol.aws.hfl.dev_tasksets.yaml` | `TaskSet` (`ecs.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `7117ca5a` (2026-11-17) |
| `ecs.cloudcontrol.aws.hfl.dev_expressgatewayservices.yaml` | `ExpressGatewayService` (`ecs.cloudcontrol.aws.hfl.dev/v1alpha1`, Cloud Control) | `cloudcontrol/config/crds/` at `7117ca5a` (2026-11-17) |
| `cloudwatchlogs.aws.hfl.dev_loggroups.yaml` | `LogGroup` (`cloudwatchlogs.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-17) |
| `efs.aws.hfl.dev_filesystems.yaml` | `FileSystem` (`efs.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-17) |
| `efs.aws.hfl.dev_mounttargets.yaml` | `MountTarget` (`efs.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-17) |
| `elasticache.aws.hfl.dev_cacheclusters.yaml` | `CacheCluster` (`elasticache.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-17) |
| `elasticache.aws.hfl.dev_replicationgroups.yaml` | `ReplicationGroup` (`elasticache.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-17) |
| `ec2.aws.hfl.dev_vpnconnections.yaml` | `VPNConnection` (`ec2.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-18) |
| `ec2.aws.hfl.dev_vpngatewayroutepropagations.yaml` | `VPNGatewayRoutePropagation` (`ec2.aws.hfl.dev/v1alpha1`) | `config/crds/bases/` at `6cb80e7c` (2026-11-18) |
