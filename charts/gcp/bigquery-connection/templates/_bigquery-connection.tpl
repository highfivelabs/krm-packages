{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* bqConnection.name: the connection's ID: names.connection in full, else
connection_id with the namePrefix (docs/design.md, "Names, labels and
tags"). */}}
{{- define "bqConnection.name" -}}
{{- include "kcc.resourceName" (list . "connection" "connection" .Values.connection_id 63) -}}
{{- end -}}

{{/* bqConnection.id: the connection's full name, the resource IAM is on. */}}
{{- define "bqConnection.id" -}}
{{- printf "projects/%s/locations/%s/connections/%s" .Values.project_id .Values.location (include "bqConnection.name" .) -}}
{{- end -}}
