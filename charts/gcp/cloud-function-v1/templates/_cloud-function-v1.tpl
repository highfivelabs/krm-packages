{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* cloudFunctionV1.prefix: fabric's local.prefix, "<prefix>-" or "". */}}
{{- define "cloudFunctionV1.prefix" -}}
{{- with .Values.prefix }}{{ . }}-{{ end -}}
{{- end -}}

{{/* cloudFunctionV1.function: the function's name: names.function in
full, else <prefix>-<name> with the namePrefix in front (63 characters, a
leading letter). */}}
{{- define "cloudFunctionV1.function" -}}
{{- include "kcc.resourceName" (list . "function" "function" (printf "%s%s" (include "cloudFunctionV1.prefix" .) .Values.name) 63 "letter") -}}
{{- end -}}

{{/* cloudFunctionV1.bucket: fabric's local.bucket. With bucket_config, the
bucket the chart creates: names.bucket in full, else
<prefix>-<bucket_name> with the namePrefix in front. Without, bucket_name
as given: it names a bucket the chart does not make. */}}
{{- define "cloudFunctionV1.bucket" -}}
{{- if .Values.bucket_config -}}
{{- include "kcc.resourceName" (list . "bucket" "bucket" (printf "%s%s" (include "cloudFunctionV1.prefix" .) .Values.bucket_name) 63) -}}
{{- else -}}
{{- .Values.bucket_name -}}
{{- end -}}
{{- end -}}

{{/* cloudFunctionV1.serviceAccount: the account ID of the service account
the chart creates: names.serviceAccount, else service_account_config.name
or name with the namePrefix in front (30 characters, a leading letter). */}}
{{- define "cloudFunctionV1.serviceAccount" -}}
{{- $c := .Values.service_account_config | default dict -}}
{{- include "kcc.resourceName" (list . "serviceAccount" "service account" ($c.name | default .Values.name) 30 "letter") -}}
{{- end -}}

{{/* cloudFunctionV1.vpcConnector: the connector's name: names.vpcConnector,
else vpc_connector_create.name or name with the namePrefix in front (25
characters, a leading letter). */}}
{{- define "cloudFunctionV1.vpcConnector" -}}
{{- include "kcc.resourceName" (list . "vpcConnector" "VPC connector" (.Values.vpc_connector_create.name | default .Values.name) 25 "letter") -}}
{{- end -}}

{{/* cloudFunctionV1.serviceAccountEmail: fabric's
local.service_account_email: the created account's email, else
service_account_config.email, else "". */}}
{{- define "cloudFunctionV1.serviceAccountEmail" -}}
{{- $c := .Values.service_account_config | default dict -}}
{{- if ne (toString $c.create) "false" -}}
{{- printf "%s@%s.iam.gserviceaccount.com" (include "cloudFunctionV1.serviceAccount" .) .Values.project_id -}}
{{- else -}}
{{- $c.email | default "" -}}
{{- end -}}
{{- end -}}
