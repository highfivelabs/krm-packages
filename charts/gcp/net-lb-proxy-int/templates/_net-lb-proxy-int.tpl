{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
Helpers of this chart, ported from cloud-foundation-fabric
modules/net-lb-proxy-int (v60.0.0), Copyright 2022 Google
LLC, Apache-2.0.
*/ -}}

{{- /* netlbpi.link: a resource link without the Compute API's URL prefix,
as Config Connector's external references take it. */ -}}
{{- define "netlbpi.link" -}}
{{- trimPrefix "https://www.googleapis.com/compute/beta/" (trimPrefix "https://www.googleapis.com/compute/v1/" (toString .)) -}}
{{- end -}}

{{- /* netlbpi.project: an entry's project_id, else the chart's. (list $ entry) */ -}}
{{- define "netlbpi.project" -}}
{{- $e := index . 1 | default dict -}}
{{- $e.project_id | default (index . 0).Values.project_id -}}
{{- end -}}

{{- /* netlbpi.negName: the cloud name of the network endpoint group of a
key, <name>-<key> with the namePrefix. (list $ key) */ -}}
{{- define "netlbpi.negName" -}}
{{- include "kcc.cloudName" (list (index . 0) "network endpoint group" (printf "%s-%s" (index . 0).Values.name (index . 1)) 63 "letter") -}}
{{- end -}}

{{- /* netlbpi.frName: the cloud name of the forwarding rule of a key: the
entry's name, else name for the key "" and <name>-<key> otherwise, with
the namePrefix. (list $ key) */ -}}
{{- define "netlbpi.frName" -}}
{{- $root := index . 0 -}}{{- $k := index . 1 -}}
{{- $frs := $root.Values.forwarding_rules_config -}}
{{- if kindIs "invalid" $frs }}{{ $frs = dict "" dict }}{{ end -}}
{{- $fr := index $frs $k | default dict -}}
{{- include "kcc.cloudName" (list $root "forwarding rule" ($fr.name | default (ternary $root.Values.name (printf "%s-%s" $root.Values.name $k) (eq $k ""))) 63 "letter") -}}
{{- end -}}

{{- /* netlbpi.hasPsc: "true" when a backend is one of the chart's PSC
NEGs; fabric then makes and attaches no health check. */ -}}
{{- define "netlbpi.hasPsc" -}}
{{- $negs := .Values.neg_configs | default dict -}}
{{- range $b := (.Values.backend_service_config | default dict).backends | default list -}}
{{- $n := index $negs $b.group | default dict -}}
{{- if not (kindIs "invalid" $n.psc) }}true{{ end -}}
{{- end -}}
{{- end -}}
