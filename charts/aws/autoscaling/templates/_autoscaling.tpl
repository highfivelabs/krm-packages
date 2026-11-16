{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The autoscaling chart's own helpers.
*/}}

{{/*
autoscaling.renames: the module's attribute names whose ACK field is not
their plain camelCase (ACK spells out acronyms and units: ID, ARN, IP,
IPv6, EBS, MiB, GiB, VCPU, DNS).
*/}}
{{- define "autoscaling.renames" -}}
{{- toJson (dict
  "accelerator_total_memory_mib" "acceleratorTotalMemoryMiB"
  "associate_carrier_ip_address" "associateCarrierIPAddress"
  "associate_public_ip_address" "associatePublicIPAddress"
  "baseline_ebs_bandwidth_mbps" "baselineEBSBandwidthMbps"
  "capacity_reservation_id" "capacityReservationID"
  "capacity_reservation_resource_group_arn" "capacityReservationResourceGroupARN"
  "disable_api_stop" "disableAPIStop"
  "disable_api_termination" "disableAPITermination"
  "enable_resource_name_dns_a_record" "enableResourceNameDNSARecord"
  "enable_resource_name_dns_aaaa_record" "enableResourceNameDNSAAAARecord"
  "group_id" "groupID"
  "host_id" "hostID"
  "host_resource_group_arn" "hostResourceGroupARN"
  "http_protocol_ipv6" "httpProtocolIPv6"
  "kms_key_id" "kmsKeyID"
  "license_configuration_arn" "licenseConfigurationARN"
  "memory_gib_per_vcpu" "memoryGiBPerVCPU"
  "memory_mib" "memoryMiB"
  "metric_dimension" "dimensions"
  "network_interface_id" "networkInterfaceID"
  "notification_target_arn" "notificationTargetARN"
  "primary_ipv6" "primaryIPv6"
  "private_dns_name_options" "privateDNSNameOptions"
  "private_ip_address" "privateIPAddress"
  "role_arn" "roleARN"
  "snapshot_id" "snapshotID"
  "step_adjustment" "stepAdjustments"
  "subnet_id" "subnetID"
  "total_local_storage_gb" "totalLocalStorageGB"
  "vcpu_count" "vCPUCount"
) -}}
{{- end -}}

{{/*
autoscaling.convert: a module value (an object, a list of them, or a
scalar) as ACK's: each attribute name in camelCase (or as
autoscaling.renames names it), null attributes left out, as JSON
{"v": value}. Lists and objects are converted all the way down.
  {{ (include "autoscaling.convert" (list .Values.cpu_options) | fromJson).v }}
*/}}
{{- define "autoscaling.convert" -}}
{{- $v := index . 0 -}}
{{- $ren := include "autoscaling.renames" . | fromJson -}}
{{- if kindIs "map" $v -}}
{{- $out := dict -}}
{{- range $k, $x := $v -}}
{{- if not (kindIs "invalid" $x) -}}
{{- $nk := index $ren $k | default (camelcase $k | untitle) -}}
{{- $_ := set $out $nk (include "autoscaling.convert" (list $x) | fromJson).v -}}
{{- end -}}
{{- end -}}
{{- toJson (dict "v" $out) -}}
{{- else if kindIs "slice" $v -}}
{{- $out := list -}}
{{- range $x := $v -}}
{{- $out = append $out (include "autoscaling.convert" (list $x) | fromJson).v -}}
{{- end -}}
{{- toJson (dict "v" $out) -}}
{{- else -}}
{{- toJson (dict "v" $v) -}}
{{- end -}}
{{- end -}}

{{/*
autoscaling.groupName: the group's name: names.autoScalingGroup, else name
(with the namePrefix). use_name_prefix's random suffix is not added (a
render must not change between plans).
*/}}
{{- define "autoscaling.groupName" -}}
{{- if not .Values.name -}}
{{- fail "autoscaling: name is required: the Auto Scaling group's name" -}}
{{- end -}}
{{- include "ack.resourceName" (list . "autoScalingGroup" "Auto Scaling group" .Values.name 255) -}}
{{- end -}}

{{/*
autoscaling.launchTemplateName: the launch template the group launches
from. With create_launch_template, the chart's: names.launchTemplate, else
coalesce(launch_template_name, name) with the namePrefix. Without, the
existing template's name, launch_template_name, as given.
*/}}
{{- define "autoscaling.launchTemplateName" -}}
{{- $v := .Values -}}
{{- if $v.create_launch_template -}}
{{- include "ack.resourceName" (list . "launchTemplate" "launch template" ($v.launch_template_name | default $v.name) 128) -}}
{{- else -}}
{{- $v.launch_template_name -}}
{{- end -}}
{{- end -}}

{{/*
autoscaling.roleName: the IAM role's name, as the module's
local.internal_iam_instance_profile_name: names.iamRole, else
coalesce(iam_instance_profile_name, iam_role_name) with the namePrefix.
*/}}
{{- define "autoscaling.roleName" -}}
{{- $v := .Values -}}
{{- $n := include "ack.resourceName" (list . "iamRole" "IAM role" ($v.iam_instance_profile_name | default $v.iam_role_name | default "") 64) -}}
{{- if not $n -}}
{{- fail "autoscaling: create_iam_instance_profile needs iam_role_name or iam_instance_profile_name (or names.iamRole): the role's name" -}}
{{- end -}}
{{- $n -}}
{{- end -}}

{{/*
autoscaling.profileName: the instance profile's name: names.instanceProfile,
else iam_role_name (the module's; unset, Terraform would generate one, so
the role's name) with the namePrefix.
*/}}
{{- define "autoscaling.profileName" -}}
{{- $v := .Values -}}
{{- if $v.iam_role_name -}}
{{- include "ack.resourceName" (list . "instanceProfile" "instance profile" $v.iam_role_name 128) -}}
{{- else if (index ($v.names | default dict) "instanceProfile") -}}
{{- include "ack.resourceName" (list . "instanceProfile" "instance profile" "" 128) -}}
{{- else -}}
{{- include "autoscaling.roleName" . -}}
{{- end -}}
{{- end -}}

{{/*
autoscaling.groupTags: the module's local.autoscaling_group_tags,
merge(tags, {Name = coalesce(instance_name, name)}, autoscaling_group_tags),
as JSON. The Name tag carries the namePrefix.
*/}}
{{- define "autoscaling.groupTags" -}}
{{- $v := .Values -}}
{{- $nameTag := include "ack.cloudName" (list . "instance Name tag" ($v.instance_name | default $v.name) 256) -}}
{{- include "ack.tags" (list . $v.tags (dict "Name" $nameTag) $v.autoscaling_group_tags) -}}
{{- end -}}
