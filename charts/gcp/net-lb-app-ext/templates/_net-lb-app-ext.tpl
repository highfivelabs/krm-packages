{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
This chart's configuration of the shared Application Load Balancer helpers
(_netlbapp.tpl): a global load balancer, EXTERNAL (classic) or
EXTERNAL_MANAGED, as cloud-foundation-fabric modules/net-lb-app-ext
(v60.0.0), Copyright 2026 Google LLC, Apache-2.0.
*/ -}}
{{- define "netlbapp.cfg" -}}
{{- toJson (dict "region" "" "scheme" (ternary "EXTERNAL_MANAGED" "EXTERNAL" (eq (toString .Values.use_classic_version) "false")) "portNameLower" true "negProject" false "negEntryDesc" true "pscDesc" true) -}}
{{- end -}}
