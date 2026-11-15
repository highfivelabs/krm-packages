{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* cas.create: "true" when the chart makes the pool (no use_pool.id, as
fabric decides). */}}
{{- define "cas.create" -}}
{{- $use := (.Values.ca_pool_config | default dict).use_pool | default dict -}}
{{- if not $use.id }}true{{ end -}}
{{- end -}}

{{/* cas.poolName: the pool's name: names.ca_pool in full, else
create_pool.name with the namePrefix; with use_pool, the last segment of
its id. */}}
{{- define "cas.poolName" -}}
{{- if include "cas.create" . -}}
{{- $cp := .Values.ca_pool_config.create_pool | default dict -}}
{{- if not $cp.name -}}
{{- fail (printf "%s: ca_pool_config needs create_pool (with a name) or use_pool (with an id)" .Chart.Name) -}}
{{- end -}}
{{- include "kcc.resourceName" (list . "ca_pool" "CA pool" $cp.name 63) -}}
{{- else -}}
{{- splitList "/" .Values.ca_pool_config.use_pool.id | last -}}
{{- end -}}
{{- end -}}

{{/* cas.poolId: the pool's full name, fabric's ca_pool_id. */}}
{{- define "cas.poolId" -}}
{{- if include "cas.create" . -}}
{{- printf "projects/%s/locations/%s/caPools/%s" .Values.project_id .Values.location (include "cas.poolName" .) -}}
{{- else -}}
{{- .Values.ca_pool_config.use_pool.id -}}
{{- end -}}
{{- end -}}

{{/* cas.bool: a key usage flag: the value set, else fabric's default
(true for the keys in the defaults dict, false otherwise). */}}
{{- define "cas.bool" -}}
{{- $m := index . 0 -}}{{- $d := index . 1 -}}{{- $f := index . 2 -}}
{{- $x := index $m $f -}}
{{- if kindIs "invalid" $x -}}{{- hasKey $d $f -}}{{- else -}}{{- $x -}}{{- end -}}
{{- end -}}
