{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

tgw.en: a boolean as the "enable" or "disable" EC2 takes for the gateway's
and attachments' options.
*/ -}}
{{- define "tgw.en" -}}
{{- if . -}}enable{{- else -}}disable{{- end -}}
{{- end -}}
