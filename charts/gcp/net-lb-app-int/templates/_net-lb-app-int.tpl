{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
This chart's configuration of the shared Application Load Balancer helpers
(_netlbapp.tpl): a regional INTERNAL_MANAGED load balancer, as
cloud-foundation-fabric modules/net-lb-app-int (v60.0.0), Copyright 2026
Google LLC, Apache-2.0.
*/ -}}
{{- define "netlbapp.cfg" -}}
{{- toJson (dict "region" .Values.region "scheme" "INTERNAL_MANAGED" "portNameLower" false "negProject" true "negEntryDesc" false "pscDesc" false) -}}
{{- end -}}
