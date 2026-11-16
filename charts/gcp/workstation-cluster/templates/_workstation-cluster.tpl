{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* workstationCluster.cluster: the cluster's ID: names.cluster in full,
else id with the namePrefix in front (63 characters, a leading letter). */}}
{{- define "workstationCluster.cluster" -}}
{{- include "kcc.resourceName" (list . "cluster" "workstation cluster" .Values.id 63 "letter") -}}
{{- end -}}

{{/* workstationCluster.clusterPath: the cluster's full name,
projects/<project>/locations/<location>/workstationClusters/<id>. */}}
{{- define "workstationCluster.clusterPath" -}}
{{- printf "projects/%s/locations/%s/workstationClusters/%s" .Values.project_id .Values.location (include "workstationCluster.cluster" .) -}}
{{- end -}}

{{/* workstationCluster.kv: a map as Config Connector's [{key, value}]
list under a key, sorted, when it has entries. */}}
{{- define "workstationCluster.kv" -}}
{{- $key := index . 0 -}}{{- $m := index . 1 -}}
{{- if $m -}}
{{ $key }}:
{{- range $k := keys $m | sortAlpha }}
  - key: {{ $k | quote }}
    value: {{ index $m $k | quote }}
{{- end }}
{{- end -}}
{{- end -}}

{{/* workstationCluster.labels: the cloud labels of one resource as JSON:
the unprefixed keys of the chart's tags and labels, then the resource's own
labels from workstation_configs (which win). */}}
{{- define "workstationCluster.labels" -}}
{{- $root := index . 0 -}}{{- $own := index . 1 | default dict -}}
{{- $cloud := dict -}}
{{- range $k, $l := (include "kcc.labels" (list $root dict) | fromJson) -}}
{{- if not (contains "/" $k) }}{{ $_ := set $cloud $k $l }}{{ end -}}
{{- end -}}
{{- range $k, $l := $own }}{{ $_ := set $cloud $k $l }}{{ end -}}
{{- toJson $cloud -}}
{{- end -}}

{{/* workstationCluster.iam: the IAM fabric manages on a workstation
configuration or a workstation, neither of which Config Connector's own
IAM kinds can reference, as config-connector-extensions' kinds
(docs/design.md, "IAM"):
  iam                      one IAMBinding (iam.gcp.hfl.dev/v1beta1) per role
  iam_bindings             one IAMBinding per entry
  iam_bindings_additive    one IAMMember (iam.gcp.hfl.dev/v1beta1) per entry
fabric's variables carry no conditions here.
Arguments (a dict):
  root      $
  ref       the resourceRef (kind and external)
  v         the configuration's or the workstation's values
  base      what the objects' names start with after the release name
  idPrefix  what their hfl.dev/ids start with
  what      how a refusal names the variables
*/}}
{{- define "workstationCluster.iam" -}}
{{- $root := .root }}{{- $v := .v }}{{- $ref := .ref }}{{- $base := .base }}{{- $idp := .idPrefix }}
{{- $iam := $v.iam | default dict }}
{{- range $role := keys $iam | sortAlpha }}
---
apiVersion: iam.gcp.hfl.dev/v1beta1
kind: IAMBinding
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (list $root.Release.Name $base "iam" $role)) "id" (printf "%siam:%s" $idp $role) "project" $root.Values.project_id) | nindent 2 }}
spec:
  resourceRef:
    {{- toYaml $ref | nindent 4 }}
  role: {{ $role | quote }}
  members:
  {{- range $m := (index $iam $role | uniq | sortAlpha) }}
    - {{ $m | quote }}
  {{- end }}
{{- end }}
{{- $bindings := $v.iam_bindings | default dict }}
{{- range $key := keys $bindings | sortAlpha }}
{{- $b := index $bindings $key }}
---
apiVersion: iam.gcp.hfl.dev/v1beta1
kind: IAMBinding
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (list $root.Release.Name $base "iam-binding" $key)) "id" (printf "%siam-binding:%s" $idp $key) "project" $root.Values.project_id) | nindent 2 }}
spec:
  resourceRef:
    {{- toYaml $ref | nindent 4 }}
  role: {{ $b.role | quote }}
  members:
  {{- range $m := ($b.members | uniq | sortAlpha) }}
    - {{ $m | quote }}
  {{- end }}
{{- end }}
{{- $additive := $v.iam_bindings_additive | default dict }}
{{- range $key := keys $additive | sortAlpha }}
{{- $b := index $additive $key }}
---
apiVersion: iam.gcp.hfl.dev/v1beta1
kind: IAMMember
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (list $root.Release.Name $base "iam-member" $key)) "id" (printf "%siam-member:%s" $idp $key) "project" $root.Values.project_id) | nindent 2 }}
spec:
  resourceRef:
    {{- toYaml $ref | nindent 4 }}
  role: {{ $b.role | quote }}
  member: {{ $b.member | quote }}
{{- end }}
{{- end -}}
