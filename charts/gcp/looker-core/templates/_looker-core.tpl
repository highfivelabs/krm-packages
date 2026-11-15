{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* lookerCore.name: the instance's ID: names.instance in full, else
fabric's local.looker_instance_name ("<prefix>-<name>", or name without a
prefix) with the namePrefix (docs/design.md, "Names, labels and tags"). */}}
{{- define "lookerCore.name" -}}
{{- $derived := .Values.name -}}
{{- with .Values.prefix }}{{ $derived = printf "%s-%s" . $derived }}{{ end -}}
{{- include "kcc.resourceName" (list . "instance" "Looker instance" $derived 63 "letter") -}}
{{- end -}}

{{/* lookerCore.time: a TimeOfDay from fabric's start_time object, with
fabric's defaults (23:00:00.0). */}}
{{- define "lookerCore.time" -}}
{{- $t := . | default dict -}}
hours: {{ ternary 23 $t.hours (kindIs "invalid" $t.hours) }}
minutes: {{ ternary 0 $t.minutes (kindIs "invalid" $t.minutes) }}
seconds: {{ ternary 0 $t.seconds (kindIs "invalid" $t.seconds) }}
nanos: {{ ternary 0 $t.nanos (kindIs "invalid" $t.nanos) }}
{{- end -}}

{{/* lookerCore.date: a Date from fabric's {year, month, day}. */}}
{{- define "lookerCore.date" -}}
year: {{ .year }}
month: {{ .month }}
day: {{ .day }}
{{- end -}}
