{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
Helpers of this chart, ported from cloud-foundation-fabric
modules/compute-vm (v60.0.0), Copyright 2024 Google LLC, Apache-2.0.
*/ -}}

{{- /* computeVm.isTemplate: "true" when create_template is set (fabric's
local.is_template). */ -}}
{{- define "computeVm.isTemplate" -}}
{{- if kindIs "map" .Values.create_template -}}true{{- end -}}
{{- end -}}

{{- /* computeVm.region: the zone's region, as fabric's local._region:
europe-west2-c and europe-west2-ai2c are in europe-west2. */ -}}
{{- define "computeVm.region" -}}
{{- regexReplaceAll "-[^-]?$|-ai[0-9][a-zA-Z]$" (toString .Values.zone) "" -}}
{{- end -}}

{{- /* computeVm.instanceName: the instance's cloud name: names.instance,
else name with the namePrefix. */ -}}
{{- define "computeVm.instanceName" -}}
{{- include "kcc.resourceName" (list . "instance" "instance" .Values.name 63 "letter") -}}
{{- end -}}

{{- /* computeVm.sameName: a name fabric gives as var.name to another
resource (the group, the schedule policy): names.<key> in full, else the
instance's name as set. */ -}}
{{- define "computeVm.sameName" -}}
{{- $root := index . 0 -}}{{- $key := index . 1 -}}
{{- $set := index ($root.Values.names | default dict) $key -}}
{{- if $set -}}{{- $set -}}{{- else -}}{{- include "computeVm.instanceName" $root -}}{{- end -}}
{{- end -}}

{{- /* computeVm.saId: the created account's ID: names.serviceAccount, else
fabric's tf-vm-<name> with the namePrefix. */ -}}
{{- define "computeVm.saId" -}}
{{- include "kcc.resourceName" (list . "serviceAccount" "service account" (printf "tf-vm-%s" .Values.name) 30 "letter") -}}
{{- end -}}

{{- /* computeVm.diskName: the cloud name of an attached disk the chart
makes (args: root, key, disk): its own name with the namePrefix, else
<instance>-<key>, as fabric's coalesce(name, "${var.name}-${key}"). */ -}}
{{- define "computeVm.diskName" -}}
{{- $root := index . 0 -}}{{- $key := index . 1 -}}{{- $d := index . 2 -}}
{{- if $d.name -}}
{{- include "kcc.cloudName" (list $root "disk" $d.name 63 "letter") -}}
{{- else -}}
{{- include "kcc.fit" (list $root "disk" (printf "%s-%s" (include "computeVm.instanceName" $root) $key) 63) -}}
{{- end -}}
{{- end -}}

{{- /* computeVm.bootDiskName: the independent boot disk's cloud name: its
own name with the namePrefix, else <instance>-boot. */ -}}
{{- define "computeVm.bootDiskName" -}}
{{- $u := .Values.boot_disk.use_independent_disk -}}
{{- if and $u $u.name -}}
{{- include "kcc.cloudName" (list . "boot disk" $u.name 63 "letter") -}}
{{- else -}}
{{- include "kcc.fit" (list . "boot disk" (printf "%s-boot" (include "computeVm.instanceName" .)) 63) -}}
{{- end -}}
{{- end -}}

{{- /* computeVm.policyName: a snapshot schedule's cloud name,
<instance>-<key>. */ -}}
{{- define "computeVm.policyName" -}}
{{- $root := index . 0 -}}{{- $key := index . 1 -}}
{{- include "kcc.fit" (list $root "resource policy" (printf "%s-%s" (include "computeVm.instanceName" $root) $key) 63) -}}
{{- end -}}

{{- /* computeVm.attachedOrder: the attached disks' keys in fabric's order
(coalesce(position, key), sorted), as JSON. */ -}}
{{- define "computeVm.attachedOrder" -}}
{{- $by := dict -}}
{{- range $k, $d := (.Values.attached_disks | default dict) -}}
{{- $_ := set $by (($d.position | default $k) | toString) $k -}}
{{- end -}}
{{- $out := list -}}
{{- range $o := keys $by | sortAlpha -}}
{{- $out = append $out (index $by $o) -}}
{{- end -}}
{{- toJson $out -}}
{{- end -}}

{{- /* computeVm.cloudLabels: the labels fabric merges into a disk's
labels (var.labels): the chart's unprefixed labels and a map of tags, as
JSON. */ -}}
{{- define "computeVm.cloudLabels" -}}
{{- $out := dict -}}
{{- if kindIs "map" .Values.tags -}}
{{- range $k, $v := .Values.tags }}{{ $_ := set $out $k (toString $v) }}{{ end -}}
{{- end -}}
{{- range $k, $v := (.Values.labels | default dict) -}}
{{- if not (contains "/" $k) }}{{ $_ := set $out $k (toString $v) }}{{ end -}}
{{- end -}}
{{- toJson $out -}}
{{- end -}}

{{- /* computeVm.sa: the instance's service account as JSON, {} for none
(service_account null): {email, scopes, created}. email is "" for the
Compute Engine default account. Scopes default as fabric's: the default
account's three, or cloud-platform and userinfo.email for any other. */ -}}
{{- define "computeVm.sa" -}}
{{- $sa := .Values.service_account -}}
{{- if kindIs "map" $sa -}}
{{- $email := "" -}}
{{- if $sa.auto_create -}}
{{- $email = printf "%s@%s.iam.gserviceaccount.com" (include "computeVm.saId" .) .Values.project_id -}}
{{- else if $sa.email -}}
{{- $email = $sa.email -}}
{{- end -}}
{{- $scopes := $sa.scopes -}}
{{- if kindIs "invalid" $scopes -}}
{{- if and (not $sa.email) (not $sa.auto_create) -}}
{{- $scopes = list "https://www.googleapis.com/auth/devstorage.read_only" "https://www.googleapis.com/auth/logging.write" "https://www.googleapis.com/auth/monitoring.write" -}}
{{- else -}}
{{- $scopes = list "https://www.googleapis.com/auth/cloud-platform" "https://www.googleapis.com/auth/userinfo.email" -}}
{{- end -}}
{{- end -}}
{{- toJson (dict "email" $email "scopes" $scopes "created" (ternary true false (eq (toString $sa.auto_create) "true"))) -}}
{{- else -}}
{}
{{- end -}}
{{- end -}}

{{- /* computeVm.scheduling: the scheduling block of an instance or
template, as YAML (fabric's scheduling {}). arg: root; ssdList true renders
localSsdRecoveryTimeout as a list (ComputeInstanceTemplate's shape). */ -}}
{{- define "computeVm.scheduling" -}}
{{- $root := index . 0 -}}{{- $ssdList := index . 1 -}}
{{- $v := $root.Values -}}
{{- $s := $v.scheduling_config | default dict -}}
{{- $spot := eq (toString $s.provisioning_model) "SPOT" -}}
{{- $ohm := $s.on_host_maintenance -}}
{{- if not $ohm -}}
{{- $ohm = ternary "TERMINATE" "MIGRATE" (or $spot (not (kindIs "invalid" $v.confidential_compute)) (kindIs "map" $v.gpu)) -}}
{{- end -}}
{{- $restart := ternary (not $spot) $s.automatic_restart (kindIs "invalid" $s.automatic_restart) -}}
automaticRestart: {{ $restart }}
{{- if or $spot (kindIs "map" $s.max_run_duration) }}
instanceTerminationAction: {{ $s.termination_action | default "STOP" | quote }}
{{- end }}
onHostMaintenance: {{ $ohm | quote }}
preemptible: {{ $spot }}
provisioningModel: {{ $s.provisioning_model | default "STANDARD" | quote }}
{{- if not (kindIs "invalid" $s.min_node_cpus) }}
minNodeCpus: {{ int64 $s.min_node_cpus }}
{{- end }}
{{- with $s.maintenance_interval }}
maintenanceInterval: {{ . | quote }}
{{- end }}
{{- with $s.max_run_duration }}
maxRunDuration:
  seconds: {{ int64 .seconds }}
  {{- if not (kindIs "invalid" .nanos) }}
  nanos: {{ int64 .nanos }}
  {{- end }}
{{- end }}
{{- with $s.local_ssd_recovery_timeout }}
localSsdRecoveryTimeout:
  {{ if $ssdList }}- {{ end }}seconds: {{ int64 .seconds }}
  {{- if not (kindIs "invalid" .nanos) }}
  {{ if $ssdList }}  {{ end }}nanos: {{ int64 .nanos }}
  {{- end }}
{{- end }}
{{- with $s.node_affinities }}
nodeAffinities:
  {{- range $k := keys . | sortAlpha }}
  {{- $a := index $s.node_affinities $k }}
  - value:
      key: {{ $k | quote }}
      operator: {{ ternary "NOT_IN" "IN" (eq (toString $a.in) "false") }}
      values: {{ toJson $a.values }}
  {{- end }}
{{- end }}
{{- end -}}

{{- /* computeVm.networkInterfaces: networkInterface entries of an instance
or template, as YAML (fabric's dynamic network_interface). arg: root. */ -}}
{{- define "computeVm.networkInterfaces" -}}
{{- range $nic := .Values.network_interfaces }}
- networkRef:
    external: {{ $nic.network | quote }}
  subnetworkRef:
    external: {{ $nic.subnetwork | quote }}
  {{- with $nic.addresses }}{{ with .internal }}
  networkIp: {{ . | quote }}
  {{- end }}{{ end }}
  {{- with $nic.nic_type }}
  nicType: {{ . | quote }}
  {{- end }}
  {{- with $nic.stack_type }}
  stackType: {{ . | quote }}
  {{- end }}
  {{- if not (kindIs "invalid" $nic.queue_count) }}
  queueCount: {{ int64 $nic.queue_count }}
  {{- end }}
  {{- if not (kindIs "invalid" $nic.internal_ipv6_prefix_length) }}
  internalIpv6PrefixLength: {{ int64 $nic.internal_ipv6_prefix_length }}
  {{- end }}
  {{- if or $nic.nat $nic.network_tier }}
  {{- $ac := dict }}
  {{- with $nic.addresses }}{{ with .external }}{{ $_ := set $ac "natIpRef" (dict "external" .) }}{{ end }}{{ end }}
  {{- with $nic.network_tier }}{{ $_ := set $ac "networkTier" . }}{{ end }}
  accessConfig:
    - {{ toJson $ac }}
  {{- end }}
  {{- if $nic.external_ipv6 }}
  ipv6AccessConfig:
    - networkTier: PREMIUM
  {{- end }}
  {{- with $nic.alias_ips }}
  aliasIpRange:
    {{- range $r := keys . | sortAlpha }}
    - subnetworkRangeName: {{ $r | quote }}
      ipCidrRange: {{ index $nic.alias_ips $r | quote }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end -}}

{{- /* computeVm.metadata: fabric's metadata map as Config Connector's
list of {key, value}, in key order. */ -}}
{{- define "computeVm.metadata" -}}
{{- range $k := keys . | sortAlpha }}
- key: {{ $k | quote }}
  value: {{ index $ $k | quote }}
{{- end }}
{{- end -}}

{{- /* computeVm.advanced: advancedMachineFeatures, as YAML, or "" when
machine_features_config sets nothing Config Connector carries. */ -}}
{{- define "computeVm.advanced" -}}
{{- $m := .Values.machine_features_config | default dict -}}
{{- if not (kindIs "invalid" $m.enable_nested_virtualization) }}
enableNestedVirtualization: {{ $m.enable_nested_virtualization }}
{{- end }}
{{- if not (kindIs "invalid" $m.threads_per_core) }}
threadsPerCore: {{ int64 $m.threads_per_core }}
{{- end }}
{{- if not (kindIs "invalid" $m.visible_core_count) }}
visibleCoreCount: {{ int64 $m.visible_core_count }}
{{- end }}
{{- end -}}

{{- /* computeVm.resourcePolicies: resourcePolicies of an instance or
template, as YAML: resource_policies as external, then the chart's
instance schedule policy by name (fabric's concat). */ -}}
{{- define "computeVm.resourcePolicies" -}}
{{- range (.Values.resource_policies | default list) }}
- external: {{ . | quote }}
{{- end }}
{{- if kindIs "map" .Values.instance_schedule }}
- name: {{ include "kcc.name" (list .Release.Name "schedule") }}
{{- end }}
{{- end -}}
