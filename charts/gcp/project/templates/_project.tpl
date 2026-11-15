{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
Ported from cloud-foundation-fabric modules/project (v60.0.0),
Copyright 2025 Google LLC, Apache-2.0.
*/ -}}

{{/* project.id: names.project in full when set; else "<prefix>-<name>", as
fabric names it, with the namePrefix in front (a project ID is global and
never reusable: 6 to 30 characters). */}}
{{- define "project.id" -}}
{{- $n := .Values.name -}}
{{- if .Values.prefix -}}{{- $n = printf "%s-%s" .Values.prefix .Values.name -}}{{- end -}}
{{- include "kcc.resourceName" (list . "project" "project ID" $n 30 "letter") -}}
{{- end -}}

{{/* project.ref: the project as a typed reference (projectRef). */}}
{{- define "project.ref" -}}
{{- if not (kindIs "invalid" .Values.project_reuse) -}}
external: {{ include "project.id" . }}
{{- else -}}
name: {{ .Release.Name }}
{{- end -}}
{{- end -}}

{{/* project.iamRef: the project as an IAM resourceRef. */}}
{{- define "project.iamRef" -}}
apiVersion: resourcemanager.cnrm.cloud.google.com/v1beta1
kind: Project
{{- if not (kindIs "invalid" .Values.project_reuse) }}
external: projects/{{ include "project.id" . }}
{{- else }}
name: {{ .Release.Name }}
{{- end }}
{{- end -}}

{{/* project.aggregations: an alert condition's aggregations under the
given key, from fabric's [{per_series_aligner, group_by_fields,
cross_series_reducer, alignment_period}]; nothing when there are none. */}}
{{- define "project.aggregations" -}}
{{- $key := index . 0 -}}{{- $list := index . 1 -}}
{{- if $list -}}
{{- $out := list -}}
{{- range $a := $list -}}
{{- $o := dict -}}
{{- with $a.alignment_period }}{{ $_ := set $o "alignmentPeriod" . }}{{ end -}}
{{- with $a.cross_series_reducer }}{{ $_ := set $o "crossSeriesReducer" . }}{{ end -}}
{{- with $a.group_by_fields }}{{ $_ := set $o "groupByFields" . }}{{ end -}}
{{- with $a.per_series_aligner }}{{ $_ := set $o "perSeriesAligner" . }}{{ end -}}
{{- $out = append $out $o -}}
{{- end -}}
{{- dict $key $out | toYaml -}}
{{- end -}}
{{- end -}}

{{/* project.trigger: an alert condition's trigger, {count, percent}. */}}
{{- define "project.trigger" -}}
{{- if . -}}
{{- $o := dict -}}
{{- if not (kindIs "invalid" .count) }}{{ $_ := set $o "count" .count }}{{ end -}}
{{- if not (kindIs "invalid" .percent) }}{{ $_ := set $o "percent" .percent }}{{ end -}}
{{- dict "trigger" $o | toYaml -}}
{{- end -}}
{{- end -}}

{{/* project.tagIAM: a tag key's or value's IAM, as the module-level IAM
variables (kcc.iam); conditions are refused, since tag keys and values
take none. Arguments: root, what (the variable path, for messages), base,
idPrefix, ref (the resourceRef), v (iam, iam_bindings,
iam_bindings_additive). */}}
{{- define "project.tagIAM" -}}
{{- $root := .root -}}
{{- range $k, $b := (.v.iam_bindings | default dict) -}}
{{- include "kcc.unsupported" (list $root (printf "%s.iam_bindings.%s.condition" $.what $k) $b.condition "tag keys and values take no IAM conditions") -}}
{{- end -}}
{{- range $k, $b := (.v.iam_bindings_additive | default dict) -}}
{{- include "kcc.unsupported" (list $root (printf "%s.iam_bindings_additive.%s.condition" $.what $k) $b.condition "tag keys and values take no IAM conditions") -}}
{{- end -}}
{{- include "kcc.iam" (dict "root" $root "base" .base "idPrefix" .idPrefix "resourceRef" .ref "v" (dict "iam" (.v.iam | default dict) "iam_bindings" (.v.iam_bindings | default dict) "iam_bindings_additive" (.v.iam_bindings_additive | default dict))) -}}
{{- end -}}

{{/* project.reuseNumber: project_reuse.attributes.number, or "" when the
chart makes the project or the number is not given. A number written
unquoted in YAML reaches the chart as a float: print it whole, never as
1.23456789012e+11. */}}
{{- define "project.reuseNumber" -}}
{{- if not (kindIs "invalid" .Values.project_reuse) -}}
{{- $n := ((.Values.project_reuse | default dict).attributes | default dict).number -}}
{{- if kindIs "float64" $n -}}{{- printf "%.0f" $n -}}
{{- else -}}{{- $n | default "" | toString -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* project.context: refuses a fabric $context lookup in a value. */}}
{{- define "project.context" -}}
{{- $root := index . 0 -}}{{- $where := index . 1 -}}{{- $value := index . 2 -}}
{{- if hasPrefix "$" (toString $value) -}}
{{- include "kcc.unsupported" (list $root "context" "x" (printf "%s %q is a fabric $context lookup; Bundt replaces them with ${...} expressions and valueFrom in the component's inputs" $where $value)) -}}
{{- end -}}
{{- end -}}
