{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* cloudDeploy.pipeline: the pipeline's name: names.pipeline in full, else
name with the namePrefix (docs/design.md, "Names, labels and tags"). */}}
{{- define "cloudDeploy.pipeline" -}}
{{- include "kcc.resourceName" (list . "pipeline" "pipeline" .Values.name 63 "letter") -}}
{{- end -}}

{{/* cloudDeploy.target: the ID of the target of one name: with the
namePrefix when the chart makes a target of that name, as given when it
does not (an existing target, or one in multi_target_target_ids that is
not the chart's). */}}
{{- define "cloudDeploy.target" -}}
{{- $root := index . 0 -}}{{- $name := index . 1 -}}
{{- $made := false -}}
{{- range $t := ($root.Values.targets | default list) -}}
{{- if and (eq $t.name $name) (or (not (kindIs "bool" $t.create_target)) $t.create_target) }}{{ $made = true }}{{ end -}}
{{- end -}}
{{- if $made -}}
{{- include "kcc.cloudName" (list $root "target" $name 63 "letter") -}}
{{- else -}}
{{- $name -}}
{{- end -}}
{{- end -}}

{{/* cloudDeploy.int: a number fabric types as a string (a time of day, a
date, retry attempts) as the integer the kinds take. */}}
{{- define "cloudDeploy.int" -}}
{{- $what := index . 1 -}}{{- $v := toString (index . 2) -}}
{{- if not (regexMatch "^[0-9]+$" $v) -}}
{{- fail (printf "%s: %s %q must be a number" (index . 0).Chart.Name $what $v) -}}
{{- end -}}
{{- $v -}}
{{- end -}}

{{/* cloudDeploy.time: a TimeOfDay (hours, minutes, seconds, nanos) from
fabric's strings, the fields set. */}}
{{- define "cloudDeploy.time" -}}
{{- $root := index . 0 -}}{{- $what := index . 1 -}}{{- $t := index . 2 -}}
{{- $out := list }}
{{- range $f := list "hours" "minutes" "seconds" "nanos" }}
{{- $x := index $t $f }}
{{- if not (kindIs "invalid" $x) }}
{{- $out = append $out (printf "%s: %s" $f (include "cloudDeploy.int" (list $root (printf "%s.%s" $what $f) $x))) }}
{{- end }}
{{- end }}
{{- join "\n" $out }}
{{- end -}}

{{/* cloudDeploy.date: a Date (day, month, year) from fabric's strings. */}}
{{- define "cloudDeploy.date" -}}
{{- $root := index . 0 -}}{{- $what := index . 1 -}}{{- $t := index . 2 -}}
{{- $out := list }}
{{- range $f := list "day" "month" "year" }}
{{- $x := index $t $f }}
{{- if not (kindIs "invalid" $x) }}
{{- $out = append $out (printf "%s: %s" $f (include "cloudDeploy.int" (list $root (printf "%s.%s" $what $f) $x))) }}
{{- end }}
{{- end }}
{{- join "\n" $out }}
{{- end -}}

{{/* cloudDeploy.strmap: a map of strings, quoted, under a key, when set. */}}
{{- define "cloudDeploy.strmap" -}}
{{- $key := index . 0 -}}{{- $m := index . 1 -}}
{{- if $m -}}
{{ $key }}:
{{- range $k := keys $m | sortAlpha }}
  {{ $k | quote }}: {{ index $m $k | quote }}
{{- end }}
{{- end -}}
{{- end -}}

{{/* cloudDeploy.strlist: a list of strings, quoted, under a key, when set. */}}
{{- define "cloudDeploy.strlist" -}}
{{- $key := index . 0 -}}{{- $l := index . 1 -}}
{{- if $l -}}
{{ $key }}:
{{- range $l }}
  - {{ . | quote }}
{{- end }}
{{- end -}}
{{- end -}}
