{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}

{{/*
ddb.tableName: the resolved AWS table name (from names.table or .Values.name).
*/}}
{{- define "ddb.tableName" -}}
{{- $v := .Values -}}
{{- if index ($v.names | default dict) "table" -}}
{{- include "ack.resourceName" (list $ "table" "table" "" 255) -}}
{{- else if $v.name -}}
{{- include "ack.cloudName" (list $ "table" $v.name 255) -}}
{{- else -}}
{{- fail "dynamodb-table: name is required: the table's name (or names.table)" -}}
{{- end -}}
{{- end -}}
