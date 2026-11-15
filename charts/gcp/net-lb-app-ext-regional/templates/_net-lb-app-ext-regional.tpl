{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
This chart's configuration of the shared Application Load Balancer helpers
(_netlbapp.tpl): a regional EXTERNAL_MANAGED load balancer with regional
health checks, as cloud-foundation-fabric modules/net-lb-app-ext-regional
(v60.0.0), Copyright 2024 Google LLC, Apache-2.0.
*/ -}}
{{- define "netlbapp.cfg" -}}
{{- toJson (dict "region" .Values.region "scheme" "EXTERNAL_MANAGED" "portNameLower" true "negProject" false "negEntryDesc" true "pscDesc" true "hcRegional" true "internetByEndpoint" true) -}}
{{- end -}}
