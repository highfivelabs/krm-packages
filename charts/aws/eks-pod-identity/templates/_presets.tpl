{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

Preset IAM policy definitions for eks-pod-identity (each Terraform preset
file ported 1:1). epi.presets returns a JSON object {p: [...]} where each
item has id, defaultName, description, and statements (in IAM JSON format).
*/}}

{{- define "epi.presets" -}}
{{- $v := .Values -}}
{{- $part := $v.partition | default "aws" -}}
{{- $prefix := $v.policy_name_prefix | default "AmazonEKS_" -}}
{{- $out := list -}}

{{- if $v.attach_amazon_managed_service_prometheus_policy -}}
{{- $stmts := list (dict "Effect" "Allow" "Action" (list "aps:RemoteWrite" "aps:QueryMetrics" "aps:GetSeries" "aps:GetLabels" "aps:GetMetricMetadata") "Resource" ($v.amazon_managed_service_prometheus_workspace_arns | default list)) -}}
{{- $out = append $out (dict "id" "amp-policy" "name" (coalesce $v.amazon_managed_service_prometheus_policy_name (printf "%sManagedServiceForPrometheus" $prefix)) "description" "Permissions for Amazon Managed Service for Prometheus" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_appmesh_controller_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "appmesh:ListVirtualRouters" "appmesh:ListVirtualServices" "appmesh:ListRoutes" "appmesh:ListGatewayRoutes" "appmesh:ListMeshes" "appmesh:ListVirtualNodes" "appmesh:ListVirtualGateways" "appmesh:DescribeMesh" "appmesh:DescribeVirtualRouter" "appmesh:DescribeRoute" "appmesh:DescribeVirtualNode" "appmesh:DescribeVirtualGateway" "appmesh:DescribeGatewayRoute" "appmesh:DescribeVirtualService" "appmesh:CreateMesh" "appmesh:CreateVirtualRouter" "appmesh:CreateVirtualGateway" "appmesh:CreateVirtualService" "appmesh:CreateGatewayRoute" "appmesh:CreateRoute" "appmesh:CreateVirtualNode" "appmesh:UpdateMesh" "appmesh:UpdateRoute" "appmesh:UpdateVirtualGateway" "appmesh:UpdateVirtualRouter" "appmesh:UpdateGatewayRoute" "appmesh:UpdateVirtualService" "appmesh:UpdateVirtualNode" "appmesh:DeleteMesh" "appmesh:DeleteRoute" "appmesh:DeleteVirtualRouter" "appmesh:DeleteGatewayRoute" "appmesh:DeleteVirtualService" "appmesh:DeleteVirtualNode" "appmesh:DeleteVirtualGateway") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole") "Resource" (list (printf "arn:%s:iam::*:role/aws-service-role/appmesh.amazonaws.com/AWSServiceRoleForAppMesh" $part)) "Condition" (dict "StringLike" (dict "iam:AWSServiceName" (list "appmesh.amazonaws.com"))))
  (dict "Effect" "Allow" "Action" (list "acm:ListCertificates" "acm:DescribeCertificate" "acm-pca:DescribeCertificateAuthority" "acm-pca:ListCertificateAuthorities") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "servicediscovery:CreateService" "servicediscovery:DeleteService" "servicediscovery:GetService" "servicediscovery:GetInstance" "servicediscovery:RegisterInstance" "servicediscovery:DeregisterInstance" "servicediscovery:ListInstances" "servicediscovery:ListNamespaces" "servicediscovery:ListServices" "servicediscovery:GetInstancesHealthStatus" "servicediscovery:UpdateInstanceCustomHealthStatus" "servicediscovery:GetOperation" "route53:GetHealthCheck" "route53:CreateHealthCheck" "route53:UpdateHealthCheck" "route53:ChangeResourceRecordSets" "route53:DeleteHealthCheck") "Resource" (list "*"))
-}}
{{- $out = append $out (dict "id" "appmesh-controller-policy" "name" (coalesce $v.appmesh_controller_policy_name (printf "%sAppMeshController" $prefix)) "description" "Permissions for AppMesh Controller" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_appmesh_envoy_proxy_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "appmesh:StreamAggregatedResources") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "acm:ExportCertificate" "acm-pca:GetCertificateAuthorityCertificate") "Resource" (list "*"))
-}}
{{- $out = append $out (dict "id" "appmesh-envoy-proxy-policy" "name" (coalesce $v.appmesh_envoy_proxy_policy_name (printf "%sAppMeshEnvoyProxy" $prefix)) "description" "Permissions for AppMesh Envoy Proxy" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_ebs_csi_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "ec2:CreateSnapshot" "ec2:AttachVolume" "ec2:DetachVolume" "ec2:ModifyVolume" "ec2:DescribeAvailabilityZones" "ec2:DescribeInstances" "ec2:DescribeInstanceTypes" "ec2:DescribeSnapshots" "ec2:DescribeTags" "ec2:DescribeVolumes" "ec2:DescribeVolumesModifications" "ec2:DescribeVolumeStatus" "ec2:EnableFastSnapshotRestores") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "ec2:CopyVolumes") "Resource" (list (printf "arn:%s:ec2:*:*:volume/vol-*" $part)))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateTags") "Resource" (list (printf "arn:%s:ec2:*:*:volume/*" $part) (printf "arn:%s:ec2:*:*:snapshot/*" $part)) "Condition" (dict "StringEquals" (dict "ec2:CreateAction" (list "CreateVolume" "CreateSnapshot" "CopyVolumes"))))
  (dict "Effect" "Allow" "Action" (list "ec2:DeleteTags") "Resource" (list (printf "arn:%s:ec2:*:*:volume/*" $part) (printf "arn:%s:ec2:*:*:snapshot/*" $part)))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateVolume" "ec2:CopyVolumes") "Resource" (list (printf "arn:%s:ec2:*:*:volume/*" $part)) "Condition" (dict "StringLike" (dict "aws:RequestTag/ebs.csi.aws.com/cluster" (list "true"))))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateVolume" "ec2:CopyVolumes") "Resource" (list (printf "arn:%s:ec2:*:*:volume/*" $part)) "Condition" (dict "StringLike" (dict "aws:RequestTag/CSIVolumeName" (list "*"))))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateVolume") "Resource" (list (printf "arn:%s:ec2:*:*:snapshot/*" $part)))
  (dict "Effect" "Allow" "Action" (list "ec2:DeleteVolume") "Resource" (list "*") "Condition" (dict "StringLike" (dict "ec2:ResourceTag/ebs.csi.aws.com/cluster" (list "true"))))
  (dict "Effect" "Allow" "Action" (list "ec2:DeleteVolume") "Resource" (list "*") "Condition" (dict "StringLike" (dict "ec2:ResourceTag/CSIVolumeName" (list "*"))))
  (dict "Effect" "Allow" "Action" (list "ec2:DeleteVolume") "Resource" (list "*") "Condition" (dict "StringLike" (dict "ec2:ResourceTag/kubernetes.io/created-for/pvc/name" (list "*"))))
  (dict "Effect" "Allow" "Action" (list "ec2:DeleteSnapshot" "ec2:LockSnapshot") "Resource" (list "*") "Condition" (dict "StringLike" (dict "ec2:ResourceTag/CSIVolumeSnapshotName" (list "*"))))
  (dict "Effect" "Allow" "Action" (list "ec2:DeleteSnapshot" "ec2:LockSnapshot") "Resource" (list "*") "Condition" (dict "StringLike" (dict "ec2:ResourceTag/ebs.csi.aws.com/cluster" (list "true"))))
-}}
{{- if $v.aws_ebs_csi_kms_arns -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "kms:CreateGrant" "kms:ListGrants" "kms:RevokeGrant") "Resource" $v.aws_ebs_csi_kms_arns "Condition" (dict "Bool" (dict "kms:GrantIsForAWSResource" (list "true")))) -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "kms:Encrypt" "kms:Decrypt" "kms:ReEncrypt*" "kms:GenerateDataKey*" "kms:DescribeKey") "Resource" $v.aws_ebs_csi_kms_arns) -}}
{{- end -}}
{{- $out = append $out (dict "id" "ebs-csi-policy" "name" (coalesce $v.aws_ebs_csi_policy_name (printf "%sEBS_CSI" $prefix)) "description" "Permissions to manage EBS volumes via the container storage interface (CSI) driver" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_efs_csi_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "ec2:DescribeAvailabilityZones" "elasticfilesystem:DescribeAccessPoints" "elasticfilesystem:DescribeFileSystems" "elasticfilesystem:DescribeMountTargets") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "elasticfilesystem:CreateAccessPoint") "Resource" (list "*") "Condition" (dict "StringLike" (dict "aws:RequestTag/efs.csi.aws.com/cluster" (list "true"))))
  (dict "Effect" "Allow" "Action" (list "elasticfilesystem:TagResource") "Resource" (list "*") "Condition" (dict "StringLike" (dict "aws:ResourceTag/efs.csi.aws.com/cluster" (list "true"))))
  (dict "Effect" "Allow" "Action" (list "elasticfilesystem:DeleteAccessPoint") "Resource" (list "*") "Condition" (dict "StringEquals" (dict "aws:ResourceTag/efs.csi.aws.com/cluster" (list "true"))))
-}}
{{- $out = append $out (dict "id" "efs-csi-policy" "name" (coalesce $v.aws_efs_csi_policy_name (printf "%sEFS_CSI" $prefix)) "description" "Permissions to manage EFS volumes via the container storage interface (CSI) driver" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_fsx_lustre_csi_policy -}}
{{- $svcRoles := $v.aws_fsx_lustre_csi_service_role_arns | default (list (printf "arn:%s:iam::*:role/aws-service-role/s3.data-source.lustre.fsx.amazonaws.com/*" $part)) -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole" "iam:AttachRolePolicy" "iam:PutRolePolicy") "Resource" $svcRoles)
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole") "Resource" (list "*") "Condition" (dict "StringLike" (dict "iam:AWSServiceName" (list "fsx.amazonaws.com"))))
  (dict "Effect" "Allow" "Action" (list "s3:ListBucket" "fsx:CreateFileSystem" "fsx:DeleteFileSystem" "fsx:DescribeFileSystems" "fsx:TagResource" "fsx:UpdateFileSystem") "Resource" (list "*"))
-}}
{{- $out = append $out (dict "id" "fsx-lustre-csi-policy" "name" (coalesce $v.aws_fsx_lustre_csi_policy_name (printf "%sFSxForLustre_CSI" $prefix)) "description" "Permissions to manage FSx Lustre volumes via the container storage interface (CSI) driver" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_gateway_controller_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "vpc-lattice:*" "ec2:DescribeVpcs" "ec2:DescribeSubnets" "ec2:DescribeTags" "ec2:DescribeSecurityGroups" "logs:CreateLogDelivery" "logs:GetLogDelivery" "logs:DescribeLogGroups" "logs:PutResourcePolicy" "logs:DescribeResourcePolicies" "logs:UpdateLogDelivery" "logs:DeleteLogDelivery" "logs:ListLogDeliveries" "tag:GetResources" "firehose:TagDeliveryStream" "s3:GetBucketPolicy" "s3:PutBucketPolicy" "tag:TagResources" "tag:UntagResources" "acm:ListCertificates") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole") "Resource" (list (printf "arn:%s:iam::*:role/aws-service-role/vpc-lattice.amazonaws.com/AWSServiceRoleForVpcLattice" $part)) "Condition" (dict "StringLike" (dict "iam:AWSServiceName" (list "vpc-lattice.amazonaws.com"))))
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole") "Resource" (list (printf "arn:%s:iam::*:role/aws-service-role/delivery.logs.amazonaws.com/AWSServiceRoleForLogDelivery" $part)) "Condition" (dict "StringLike" (dict "iam:AWSServiceName" (list "delivery.logs.amazonaws.com"))))
-}}
{{- $out = append $out (dict "id" "gateway-controller-policy" "name" (coalesce $v.aws_gateway_controller_policy_name (printf "%sGatewayController" $prefix)) "description" "Permissions for the AWS Gateway Controller" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_lb_controller_policy -}}
{{- $tgResources := list (printf "arn:%s:elasticloadbalancing:*:*:targetgroup/*/*" $part) (printf "arn:%s:elasticloadbalancing:*:*:loadbalancer/net/*/*" $part) (printf "arn:%s:elasticloadbalancing:*:*:loadbalancer/app/*/*" $part) -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole") "Resource" (list "*") "Condition" (dict "StringEquals" (dict "iam:AWSServiceName" (list "elasticloadbalancing.amazonaws.com"))))
  (dict "Effect" "Allow" "Action" (list "ec2:DescribeAccountAttributes" "ec2:DescribeAddresses" "ec2:DescribeAvailabilityZones" "ec2:DescribeInternetGateways" "ec2:DescribeVpcs" "ec2:DescribeVpcPeeringConnections" "ec2:DescribeSubnets" "ec2:DescribeSecurityGroups" "ec2:DescribeInstances" "ec2:DescribeNetworkInterfaces" "ec2:DescribeTags" "ec2:GetCoipPoolUsage" "ec2:DescribeCoipPools" "ec2:GetSecurityGroupsForVpc" "ec2:DescribeIpamPools" "ec2:DescribeRouteTables" "elasticloadbalancing:DescribeLoadBalancers" "elasticloadbalancing:DescribeLoadBalancerAttributes" "elasticloadbalancing:DescribeListeners" "elasticloadbalancing:DescribeListenerCertificates" "elasticloadbalancing:DescribeSSLPolicies" "elasticloadbalancing:DescribeRules" "elasticloadbalancing:DescribeTargetGroups" "elasticloadbalancing:DescribeTargetGroupAttributes" "elasticloadbalancing:DescribeTargetHealth" "elasticloadbalancing:DescribeTags" "elasticloadbalancing:DescribeTrustStores" "elasticloadbalancing:DescribeListenerAttributes" "elasticloadbalancing:DescribeCapacityReservation") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "cognito-idp:DescribeUserPoolClient" "acm:ListCertificates" "acm:DescribeCertificate" "iam:ListServerCertificates" "iam:GetServerCertificate" "waf-regional:GetWebACL" "waf-regional:GetWebACLForResource" "waf-regional:AssociateWebACL" "waf-regional:DisassociateWebACL" "wafv2:GetWebACL" "wafv2:GetWebACLForResource" "wafv2:AssociateWebACL" "wafv2:DisassociateWebACL" "wafv2:ListWebACLs" "shield:GetSubscriptionState" "shield:DescribeProtection" "shield:CreateProtection" "shield:DeleteProtection") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "ec2:AuthorizeSecurityGroupIngress" "ec2:RevokeSecurityGroupIngress") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateSecurityGroup") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateTags") "Resource" (list (printf "arn:%s:ec2:*:*:security-group/*" $part)) "Condition" (dict "StringEquals" (dict "ec2:CreateAction" (list "CreateSecurityGroup")) "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "ec2:CreateTags" "ec2:DeleteTags") "Resource" (list (printf "arn:%s:ec2:*:*:security-group/*" $part)) "Condition" (dict "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "true") "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "ec2:AuthorizeSecurityGroupIngress" "ec2:RevokeSecurityGroupIngress" "ec2:DeleteSecurityGroup") "Resource" (list "*") "Condition" (dict "Null" (dict "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:CreateLoadBalancer" "elasticloadbalancing:CreateTargetGroup") "Resource" (list "*") "Condition" (dict "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:CreateListener" "elasticloadbalancing:DeleteListener" "elasticloadbalancing:CreateRule" "elasticloadbalancing:DeleteRule") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:AddTags" "elasticloadbalancing:RemoveTags") "Resource" $tgResources "Condition" (dict "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "true") "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:AddTags" "elasticloadbalancing:RemoveTags") "Resource" (list (printf "arn:%s:elasticloadbalancing:*:*:listener/net/*/*/*" $part) (printf "arn:%s:elasticloadbalancing:*:*:listener/app/*/*/*" $part) (printf "arn:%s:elasticloadbalancing:*:*:listener-rule/net/*/*/*" $part) (printf "arn:%s:elasticloadbalancing:*:*:listener-rule/app/*/*/*" $part)))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:ModifyLoadBalancerAttributes" "elasticloadbalancing:SetIpAddressType" "elasticloadbalancing:SetSecurityGroups" "elasticloadbalancing:SetSubnets" "elasticloadbalancing:DeleteLoadBalancer" "elasticloadbalancing:ModifyTargetGroup" "elasticloadbalancing:ModifyTargetGroupAttributes" "elasticloadbalancing:DeleteTargetGroup" "elasticloadbalancing:ModifyListenerAttributes" "elasticloadbalancing:ModifyCapacityReservation" "elasticloadbalancing:ModifyIpPools") "Resource" (list "*") "Condition" (dict "Null" (dict "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:AddTags") "Resource" $tgResources "Condition" (dict "StringEquals" (dict "elasticloadbalancing:CreateAction" (list "CreateTargetGroup" "CreateLoadBalancer")) "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "false"))))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:RegisterTargets" "elasticloadbalancing:DeregisterTargets") "Resource" (list (printf "arn:%s:elasticloadbalancing:*:*:targetgroup/*/*" $part)))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:SetWebAcl" "elasticloadbalancing:ModifyListener" "elasticloadbalancing:AddListenerCertificates" "elasticloadbalancing:RemoveListenerCertificates" "elasticloadbalancing:ModifyRule" "elasticloadbalancing:SetRulePriorities") "Resource" (list "*"))
-}}
{{- $out = append $out (dict "id" "lb-controller-policy" "name" (coalesce $v.aws_lb_controller_policy_name (printf "%sLoadBalancerController" $prefix)) "description" "Permissions for AWS Load Balancer Controller" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_global_accelerator_controller_policy -}}
{{- $gaResources := list (printf "arn:%s:globalaccelerator::*:accelerator/*" $part) (printf "arn:%s:globalaccelerator::*:accelerator/*/listener/*" $part) (printf "arn:%s:globalaccelerator::*:accelerator/*/listener/*/endpoint-group/*" $part) -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "iam:CreateServiceLinkedRole") "Resource" (list "*") "Condition" (dict "StringEquals" (dict "iam:AWSServiceName" (list "globalaccelerator.amazonaws.com"))))
  (dict "Effect" "Allow" "Action" (list "globalaccelerator:ListAccelerators" "globalaccelerator:ListEndpointGroups" "globalaccelerator:ListListeners" "globalaccelerator:ListTagsForResource" "ec2:DescribeRegions" "tag:GetResources") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "globalaccelerator:DescribeAccelerator" "globalaccelerator:DescribeEndpointGroup" "globalaccelerator:DescribeListener") "Resource" $gaResources "Condition" (dict "Null" (dict "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false")) "StringEquals" (dict "aws:ResourceTag/aga.k8s.aws/resource" (list "GlobalAccelerator"))))
  (dict "Effect" "Allow" "Action" (list "globalaccelerator:CreateAccelerator") "Resource" (list "*") "Condition" (dict "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "false")) "StringEquals" (dict "aws:RequestTag/aga.k8s.aws/resource" (list "GlobalAccelerator"))))
  (dict "Effect" "Allow" "Action" (list "globalaccelerator:UpdateAccelerator" "globalaccelerator:DeleteAccelerator" "globalaccelerator:CreateListener" "globalaccelerator:UpdateListener" "globalaccelerator:DeleteListener" "globalaccelerator:CreateEndpointGroup" "globalaccelerator:UpdateEndpointGroup" "globalaccelerator:DeleteEndpointGroup" "globalaccelerator:AddEndpoints" "globalaccelerator:RemoveEndpoints") "Resource" $gaResources "Condition" (dict "Null" (dict "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false")) "StringEquals" (dict "aws:ResourceTag/aga.k8s.aws/resource" (list "GlobalAccelerator"))))
  (dict "Effect" "Allow" "Action" (list "globalaccelerator:TagResource" "globalaccelerator:UntagResource") "Resource" (list (printf "arn:%s:globalaccelerator::*:accelerator/*" $part)) "Condition" (dict "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "true") "aws:ResourceTag/elbv2.k8s.aws/cluster" (list "false")) "StringEquals" (dict "aws:ResourceTag/aga.k8s.aws/resource" (list "GlobalAccelerator"))))
  (dict "Effect" "Allow" "Action" (list "globalaccelerator:TagResource") "Resource" (list (printf "arn:%s:globalaccelerator::*:accelerator/*" $part)) "Condition" (dict "Null" (dict "aws:RequestTag/elbv2.k8s.aws/cluster" (list "false")) "StringEquals" (dict "aws:RequestTag/aga.k8s.aws/resource" (list "GlobalAccelerator"))))
-}}
{{- $out = append $out (dict "id" "global-accelerator-controller-policy" "name" (coalesce $v.aws_global_accelerator_controller_policy_name (printf "%sGlobalAcceleratorController" $prefix)) "description" "Permissions for AWS Global Accelerator Controller" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_lb_controller_targetgroup_binding_only_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "ec2:DescribeVpcs" "ec2:DescribeSecurityGroups" "ec2:DescribeInstances" "ec2:AuthorizeSecurityGroupIngress" "ec2:RevokeSecurityGroupIngress" "elasticloadbalancing:DescribeTargetGroups" "elasticloadbalancing:DescribeTargetHealth") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "elasticloadbalancing:ModifyTargetGroup" "elasticloadbalancing:ModifyTargetGroupAttributes" "elasticloadbalancing:RegisterTargets" "elasticloadbalancing:DeregisterTargets") "Resource" ($v.aws_lb_controller_targetgroup_arns | default list))
-}}
{{- $out = append $out (dict "id" "lb-controller-targetgroup-only-policy" "name" (coalesce $v.aws_lb_controller_targetgroup_only_policy_name (printf "%sLoadBalancerControllerTargetGroupOnly" $prefix)) "description" "Permissions for AWS Load Balancer Controller with TargetGroup binding only" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_node_termination_handler_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "autoscaling:CompleteLifecycleAction" "autoscaling:DescribeAutoScalingInstances" "autoscaling:DescribeTags" "ec2:DescribeInstances") "Resource" (list "*"))
  (dict "Effect" "Allow" "Action" (list "sqs:DeleteMessage" "sqs:ReceiveMessage") "Resource" ($v.aws_node_termination_handler_sqs_queue_arns | default list))
-}}
{{- $out = append $out (dict "id" "node-termination-handler-policy" "name" (coalesce $v.aws_node_termination_handler_policy_name (printf "%sNodeTerminationHandler" $prefix)) "description" "Permissions for Node Termination Handler" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_privateca_issuer_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "acm-pca:DescribeCertificateAuthority" "acm-pca:GetCertificate" "acm-pca:IssueCertificate") "Resource" ($v.aws_privateca_issuer_acmca_arns | default list))
-}}
{{- $out = append $out (dict "id" "privateca-issuer-policy" "name" (coalesce $v.aws_privateca_issuer_policy_name (printf "%sPrivateCAIssuer" $prefix)) "description" "Permissions for AWS Private CA Issuer" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_aws_vpc_cni_policy -}}
{{- $stmts := list -}}
{{- if $v.aws_vpc_cni_enable_cloudwatch_logs -}}
{{- $stmts = append $stmts (dict "Sid" "CloudWatchLogs" "Effect" "Allow" "Action" (list "logs:DescribeLogGroups" "logs:CreateLogGroup" "logs:CreateLogStream" "logs:PutLogEvents") "Resource" (list "*")) -}}
{{- end -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "ec2:CreateTags") "Resource" (list (printf "arn:%s:ec2:*:*:network-interface/*" $part))) -}}
{{- if $v.aws_vpc_cni_enable_ipv4 -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "ec2:AssignPrivateIpAddresses" "ec2:AttachNetworkInterface" "ec2:CreateNetworkInterface" "ec2:DeleteNetworkInterface" "ec2:DescribeInstances" "ec2:DescribeTags" "ec2:DescribeNetworkInterfaces" "ec2:DescribeInstanceTypes" "ec2:DescribeSecurityGroups" "ec2:DescribeSubnets" "ec2:DetachNetworkInterface" "ec2:ModifyNetworkInterfaceAttribute" "ec2:UnassignPrivateIpAddresses") "Resource" (list "*")) -}}
{{- end -}}
{{- if $v.aws_vpc_cni_enable_ipv6 -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "ec2:AssignIpv6Addresses" "ec2:DescribeInstances" "ec2:DescribeTags" "ec2:DescribeNetworkInterfaces" "ec2:DescribeInstanceTypes" "ec2:DescribeSecurityGroups" "ec2:DescribeSubnets") "Resource" (list "*")) -}}
{{- end -}}
{{- $out = append $out (dict "id" "vpc-cni-policy" "name" (coalesce $v.aws_vpc_cni_policy_name (printf "%sVPC_CNI" $prefix)) "description" "Permissions for Amazon VPC CNI" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_cert_manager_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "route53:GetChange") "Resource" (list (printf "arn:%s:route53:::change/*" $part)))
  (dict "Effect" "Allow" "Action" (list "route53:ChangeResourceRecordSets" "route53:ListResourceRecordSets") "Resource" ($v.cert_manager_hosted_zone_arns | default list))
  (dict "Effect" "Allow" "Action" (list "route53:ListHostedZonesByName") "Resource" (list "*"))
-}}
{{- $out = append $out (dict "id" "cert-manager-policy" "name" (coalesce $v.cert_manager_policy_name (printf "%sCertManager" $prefix)) "description" "Permissions for Cert Manager" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_cluster_autoscaler_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "autoscaling:DescribeAutoScalingGroups" "autoscaling:DescribeAutoScalingInstances" "autoscaling:DescribeLaunchConfigurations" "autoscaling:DescribeScalingActivities" "autoscaling:DescribeTags" "ec2:DescribeLaunchTemplateVersions" "ec2:DescribeInstanceTypes" "eks:DescribeNodegroup" "ec2:DescribeImages" "ec2:GetInstanceTypesFromInstanceRequirements") "Resource" (list "*"))
-}}
{{- range $cname := ($v.cluster_autoscaler_cluster_names | default list | uniq) -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "autoscaling:SetDesiredCapacity" "autoscaling:TerminateInstanceInAutoScalingGroup") "Resource" (list "*") "Condition" (dict "StringEquals" (dict (printf "autoscaling:ResourceTag/kubernetes.io/cluster/%s" $cname) (list "owned")))) -}}
{{- end -}}
{{- $out = append $out (dict "id" "cluster-autoscaler-policy" "name" (coalesce $v.cluster_autoscaler_policy_name (printf "%sClusterAutoscaler" $prefix)) "description" "Permissions for Cluster Autoscaler" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_external_dns_policy -}}
{{- $stmts := list
  (dict "Effect" "Allow" "Action" (list "route53:ChangeResourceRecordSets") "Resource" ($v.external_dns_hosted_zone_arns | default list))
  (dict "Effect" "Allow" "Action" (list "route53:ListHostedZones" "route53:ListResourceRecordSets" "route53:ListTagsForResource" "route53:ListTagsForResources") "Resource" (list "*"))
-}}
{{- $out = append $out (dict "id" "external-dns-policy" "name" (coalesce $v.external_dns_policy_name (printf "%sExternalDNS" $prefix)) "description" "Permissions for External DNS" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_external_secrets_policy -}}
{{- $stmts := list -}}
{{- if $v.external_secrets_ssm_parameter_arns -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "ssm:DescribeParameters") "Resource" (list "*")) -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "ssm:GetParameter" "ssm:GetParameters" "ssm:GetParametersByPath") "Resource" $v.external_secrets_ssm_parameter_arns) -}}
{{- end -}}
{{- if $v.external_secrets_secrets_manager_arns -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "secretsmanager:ListSecrets" "secretsmanager:BatchGetSecretValue") "Resource" (list "*")) -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "secretsmanager:GetResourcePolicy" "secretsmanager:GetSecretValue" "secretsmanager:DescribeSecret" "secretsmanager:ListSecretVersionIds") "Resource" $v.external_secrets_secrets_manager_arns) -}}
{{- end -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "kms:Decrypt") "Resource" ($v.external_secrets_kms_key_arns | default (list (printf "arn:%s:kms:*:*:key/*" $part)))) -}}
{{- if and $v.external_secrets_secrets_manager_arns $v.external_secrets_create_permission -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "secretsmanager:CreateSecret" "secretsmanager:PutSecretValue" "secretsmanager:TagResource") "Resource" $v.external_secrets_secrets_manager_arns) -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "secretsmanager:DeleteSecret") "Resource" $v.external_secrets_secrets_manager_arns "Condition" (dict "StringEquals" (dict "secretsmanager:ResourceTag/managed-by" (list "external-secrets")))) -}}
{{- end -}}
{{- $out = append $out (dict "id" "external-secrets-policy" "name" (coalesce $v.external_secrets_policy_name (printf "%sExternalSecrets" $prefix)) "description" "Permissions for External Secrets" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_mountpoint_s3_csi_policy -}}
{{- $stmts := list
  (dict "Sid" "MountpointFullBucketAccess" "Effect" "Allow" "Action" (list "s3:ListBucket") "Resource" ($v.mountpoint_s3_csi_bucket_arns | default (list (printf "arn:%s:s3:::*" $part))))
  (dict "Sid" "MountpointFullObjectAccess" "Effect" "Allow" "Action" (list "s3:GetObject" "s3:PutObject" "s3:AbortMultipartUpload" "s3:DeleteObject") "Resource" ($v.mountpoint_s3_csi_bucket_path_arns | default list))
-}}
{{- $out = append $out (dict "id" "mountpoint-s3-csi-policy" "name" (coalesce $v.mountpoint_s3_csi_policy_name (printf "%sMountpointS3_CSI" $prefix)) "description" "Permissions for Mountpoint S3 CSI Driver" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_pganalyze_policy -}}
{{- if not $v.account_id -}}
{{- fail "eks-pod-identity: account_id is required when attach_pganalyze_policy is true (the chart cannot look it up): values.yaml" -}}
{{- end -}}
{{- if not $v.region -}}
{{- fail "eks-pod-identity: region is required when attach_pganalyze_policy is true (the chart cannot look it up): values.yaml" -}}
{{- end -}}
{{- $stmts := list
  (dict "Sid" "CloudwatchGetMetrics" "Effect" "Allow" "Action" (list "cloudwatch:GetMetricStatistics") "Resource" (list "*"))
  (dict "Sid" "CloudwatchLogEvents" "Effect" "Allow" "Action" (list "logs:GetLogEvents") "Resource" (list (printf "arn:%s:logs:%s:%s:log-group:RDSOSMetrics:log-stream:*" $part $v.region $v.account_id)))
  (dict "Sid" "RDSDescribeParameters" "Effect" "Allow" "Action" (list "rds:DescribeDBParameters") "Resource" (list (printf "arn:%s:rds:%s:%s:pg:*" $part $v.region $v.account_id)))
  (dict "Sid" "RDSInstanceDescribe" "Effect" "Allow" "Action" (list "rds:DescribeDBInstances" "rds:DownloadDBLogFilePortion" "rds:DescribeDBLogFiles") "Resource" (list (printf "arn:%s:rds:%s:%s:db:*" $part $v.region $v.account_id)))
  (dict "Sid" "RDSClusterDescribe" "Effect" "Allow" "Action" (list "rds:DescribeDBClusters") "Resource" (list (printf "arn:%s:rds:%s:%s:cluster:*" $part $v.region $v.account_id)))
-}}
{{- $out = append $out (dict "id" "pganalyze-policy" "name" (coalesce $v.pganalyze_policy_name (printf "%sPGAnalyze" $prefix)) "description" "Permissions for PGAnalyze" "statements" $stmts) -}}
{{- end -}}

{{- if $v.attach_velero_policy -}}
{{- $stmts := list
  (dict "Sid" "Ec2ReadWrite" "Effect" "Allow" "Action" (list "ec2:DescribeVolumes" "ec2:DescribeSnapshots" "ec2:CreateTags" "ec2:CreateVolume" "ec2:CreateSnapshot" "ec2:DeleteSnapshot") "Resource" (list "*"))
  (dict "Sid" "S3ReadWrite" "Effect" "Allow" "Action" (list "s3:GetObject" "s3:DeleteObject" "s3:PutObject" "s3:PutObjectTagging" "s3:AbortMultipartUpload" "s3:ListMultipartUploadParts") "Resource" ($v.velero_s3_bucket_path_arns | default list))
  (dict "Sid" "S3List" "Effect" "Allow" "Action" (list "s3:ListBucket") "Resource" ($v.velero_s3_bucket_arns | default (list (printf "arn:%s:s3:::*" $part))))
-}}
{{- if $v.velero_kms_arns -}}
{{- $stmts = append $stmts (dict "Effect" "Allow" "Action" (list "kms:Encrypt" "kms:Decrypt" "kms:ReEncrypt*" "kms:GenerateDataKey*" "kms:DescribeKey" "kms:CreateGrant" "kms:ListGrants" "kms:RevokeGrant" "kms:RetireGrant" "kms:ListRetirableGrants") "Resource" $v.velero_kms_arns) -}}
{{- end -}}
{{- $out = append $out (dict "id" "velero-policy" "name" (coalesce $v.velero_policy_name (printf "%sVelero" $prefix)) "description" "Permissions for Velero" "statements" $stmts) -}}
{{- end -}}

{{- toJson (dict "p" $out) -}}
{{- end -}}
