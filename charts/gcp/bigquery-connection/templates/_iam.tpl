{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/*
bqConnection.iam: the IAM fabric manages on one resource that Config Connector's
own IAM kinds cannot reference, as config-connector-extensions' kinds
(docs/design.md, "IAM"):
  iam, iam_by_principals   one IAMBinding (iam.gcp.hfl.dev/v1beta1) per
                           role, the two merged as fabric merges them
  iam_bindings             one IAMBinding per entry
  iam_bindings_additive    one IAMMember (iam.gcp.hfl.dev/v1alpha1, so
                           allow_alpha) per entry
Arguments (a dict):
  root        $
  ref         the resourceRef (a dict: kind, and external or name)
  project     the project annotation
  v           a dict with iam, iam_by_principals, iam_bindings and
              iam_bindings_additive (missing ones are empty)
  conditions  true to send iam_bindings' and iam_bindings_additive's
              conditions (fabric sends them)
  base        what the objects' names start with after the release name
              ("" for the chart's core resource)
  idPrefix    what their hfl.dev/ids start with ("" for the core resource)
  what        how a refusal names the variables ("" for the top level)
*/}}
{{- define "bqConnection.iam" -}}
{{- $root := .root }}{{- $v := .v }}{{- $ref := .ref }}{{- $cond := .conditions }}
{{- $base := .base | default "" }}{{- $idp := .idPrefix | default "" }}
{{- $roles := dict }}
{{- range $role, $members := (index $v "iam" | default dict) }}
{{- $_ := set $roles $role (concat (get $roles $role | default list) $members) }}
{{- end }}
{{- range $principal, $prs := (index $v "iam_by_principals" | default dict) }}
{{- range $role := $prs }}
{{- $_ := set $roles $role (append (get $roles $role | default list) $principal) }}
{{- end }}
{{- end }}
{{- range $role := keys $roles | sortAlpha }}
---
apiVersion: iam.gcp.hfl.dev/v1beta1
kind: IAMBinding
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (compact (list $root.Release.Name $base "iam" $role))) "id" (printf "%siam:%s" $idp $role) "project" $.project) | nindent 2 }}
spec:
  resourceRef:
    {{- toYaml $ref | nindent 4 }}
  role: {{ $role | quote }}
  members:
  {{- range $m := (index $roles $role | uniq | sortAlpha) }}
    - {{ $m | quote }}
  {{- end }}
{{- end }}
{{- $bindings := index $v "iam_bindings" | default dict }}
{{- range $key := keys $bindings | sortAlpha }}
{{- $b := index $bindings $key }}
---
apiVersion: iam.gcp.hfl.dev/v1beta1
kind: IAMBinding
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (compact (list $root.Release.Name $base "iam-binding" $key))) "id" (printf "%siam-binding:%s" $idp $key) "project" $.project) | nindent 2 }}
spec:
  resourceRef:
    {{- toYaml $ref | nindent 4 }}
  role: {{ $b.role | quote }}
  members:
  {{- range $m := ($b.members | uniq | sortAlpha) }}
    - {{ $m | quote }}
  {{- end }}
  {{- if $cond }}
  {{- with $b.condition }}
  condition:
    {{- include "kcc.condition" . | nindent 4 }}
  {{- end }}
  {{- end }}
{{- end }}
{{- $additive := index $v "iam_bindings_additive" | default dict }}
{{- include "kcc.alpha" (list $root (printf "%siam_bindings_additive" (.what | default "")) $additive "IAMMember (config-connector-extensions)") }}
{{- range $key := keys $additive | sortAlpha }}
{{- $b := index $additive $key }}
---
apiVersion: iam.gcp.hfl.dev/v1alpha1
kind: IAMMember
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (compact (list $root.Release.Name $base "iam-member" $key))) "id" (printf "%siam-member:%s" $idp $key) "project" $.project) | nindent 2 }}
spec:
  resourceRef:
    {{- toYaml $ref | nindent 4 }}
  role: {{ $b.role | quote }}
  member: {{ $b.member | quote }}
  {{- if $cond }}
  {{- with $b.condition }}
  condition:
    {{- include "kcc.condition" . | nindent 4 }}
  {{- end }}
  {{- end }}
{{- end }}
{{- end -}}
