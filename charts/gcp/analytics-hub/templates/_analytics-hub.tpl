{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* analyticsHub.exchange: the data exchange's ID: names.data_exchange in
full, else fabric's <prefix>_<name>, with the namePrefix in front as
<namePrefix>_ (its hyphens as underscores: an exchange ID is letters,
digits and underscores). */}}
{{- define "analyticsHub.exchange" -}}
{{- $v := .Values -}}
{{- $set := index ($v.names | default dict) "data_exchange" | default "" -}}
{{- if $set -}}
{{- $set -}}
{{- else -}}
{{- $id := ternary (printf "%s_%s" $v.prefix $v.name) $v.name (not (empty $v.prefix)) -}}
{{- $p := include "kcc.namePrefix" . | trimSuffix "-" | replace "-" "_" -}}
{{- if $p -}}
{{- $id = printf "%s_%s" $p $id -}}
{{- if gt (len $id) 100 -}}
{{- fail (printf "%s: data exchange name %q is %d characters, more than the 100 the cloud allows; namePrefix %q is %d of them (with its \"_\"): shorten the namePrefix or the name" .Chart.Name $id (len $id) $v.namePrefix (add1 (len $v.namePrefix))) -}}
{{- end -}}
{{- end -}}
{{- if not (regexMatch "^[A-Za-z0-9_]+$" $id) -}}
{{- fail (printf "%s: data exchange ID %q may contain only letters, digits and underscores" .Chart.Name $id) -}}
{{- end -}}
{{- $id -}}
{{- end -}}
{{- end -}}

{{/* analyticsHub.id: the data exchange's full name. */}}
{{- define "analyticsHub.id" -}}
{{- printf "projects/%s/locations/%s/dataExchanges/%s" .Values.project_id .Values.region (include "analyticsHub.exchange" .) -}}
{{- end -}}
