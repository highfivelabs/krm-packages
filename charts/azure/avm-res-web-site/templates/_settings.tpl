{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
Ported from terraform-azurerm-avm-res-web-site locals.app_settings.tf,
main.config.tf, modules/config_appsettings, modules/config_connectionstrings
and modules/slot (v0.24.1), Copyright (c) Microsoft Corporation, MIT.

App settings and connection strings whose values are secret material, as
aso-extensions' SitesConfigAppSettings and SitesConfigConnectionStrings
(web.azure.hfl.dev/v20260715; RECWEB, the builder's decision of
2026-11-15): each secret value a reference to a Kubernetes Secret
({secretName, key}, docs/secrets.md), never a value in the component's
inputs.

A setting is an entry of a map from its name to {"value": <text>} (plain)
or {"ref": {name, key}, "template": <text holding $(value)>} (from a
Secret; template only when the value is part of a longer string, as the
module's storage connection strings are). When an app's settings hold a
reference, all of them are the app's SitesConfigAppSettings (one writer:
the Site's or slot's siteConfig.appSettings is then left out); otherwise
they stay inline, as before.
*/ -}}

{{/*
webcfg.secretSetting: {"ref": ..., "template": ...} as JSON for a value that
must be a reference to a Kubernetes Secret, or "" when it is null.
  (list $ "<variable path>" <value> "<template or empty>")
*/}}
{{- define "webcfg.secretSetting" -}}
{{- $root := index . 0 -}}{{- $path := index . 1 -}}{{- $v := index . 2 -}}{{- $tmpl := index . 3 -}}
{{- $ref := include "aso.secretRef" (list $root $path $v) -}}
{{- if $ref -}}
{{- $e := dict "ref" (fromJson $ref) -}}
{{- if $tmpl }}{{ $_ := set $e "template" $tmpl }}{{ end -}}
{{- toJson $e -}}
{{- end -}}
{{- end -}}

{{/*
webcfg.hasRef: "true" when an entry of the settings map is a reference.
*/}}
{{- define "webcfg.hasRef" -}}
{{- $has := false -}}
{{- range $k, $e := . -}}{{- if hasKey $e "ref" }}{{ $has = true }}{{ end -}}{{- end -}}
{{- if $has }}true{{ end -}}
{{- end -}}

{{/*
webcfg.plain: the plain map (name to text) of a settings map whose entries
are all plain, for the inline siteConfig.appSettings.
*/}}
{{- define "webcfg.plain" -}}
{{- $o := dict -}}
{{- range $k, $e := . }}{{ $_ := set $o $k $e.value }}{{ end -}}
{{- toJson $o -}}
{{- end -}}

{{/*
webcfg.appSettings: one SitesConfigAppSettings, the app's whole set.
  root, name, id, abandon   as aso.metadata
  owner                     {name} for the Site, {name, group, kind} for a slot
  settings                  the settings map
*/}}
{{- define "webcfg.appSettings" -}}
{{- $items := list -}}
{{- range $k := keys .settings | sortAlpha -}}
{{- $e := index $.settings $k -}}
{{- if hasKey $e "ref" -}}
{{- $i := dict "name" $k "valueFromSecret" $e.ref -}}
{{- with $e.template }}{{ $_ := set $i "template" . }}{{ end -}}
{{- $items = append $items $i -}}
{{- else -}}
{{- $items = append $items (dict "name" $k "value" (toString $e.value)) -}}
{{- end -}}
{{- end }}
---
apiVersion: web.azure.hfl.dev/v20260715
kind: SitesConfigAppSettings
metadata:
  {{- include "aso.metadata" (dict "root" .root "name" .name "id" .id "abandon" .abandon) | nindent 2 }}
spec:
  owner:
    {{- toYaml .owner | nindent 4 }}
  settings:
    {{- toYaml $items | nindent 4 }}
{{- end -}}

{{/*
webcfg.connectionStrings: one SitesConfigConnectionStrings from a
connection_strings map (modules/config_connectionstrings: the name is the
entry's name, else its key), each value a reference to a Kubernetes Secret.
  root, name, id, abandon, owner   as webcfg.appSettings
  path                             the variable's path, for messages
  v                                the connection_strings map
*/}}
{{- define "webcfg.connectionStrings" -}}
{{- $root := .root -}}
{{- $byName := dict -}}
{{- range $k := keys .v | sortAlpha -}}
{{- $c := index $.v $k | default dict -}}
{{- $p := printf "%s.%s" $.path $k -}}
{{- $cn := $c.name | default $k -}}
{{- if hasKey $byName $cn }}{{ fail (printf "%s: %s: connection string name %q is given twice" $root.Chart.Name $p $cn) }}{{ end -}}
{{- if not $c.type }}{{ fail (printf "%s: %s.type is required: Azure takes a connection string only with its type (MySql, SQLServer, SQLAzure, Custom, NotificationHub, ServiceBus, EventHub, ApiHub, DocDb, RedisCache or PostgreSQL)" $root.Chart.Name $p) }}{{ end -}}
{{- if not (has $c.type (list "MySql" "SQLServer" "SQLAzure" "Custom" "NotificationHub" "ServiceBus" "EventHub" "ApiHub" "DocDb" "RedisCache" "PostgreSQL")) -}}
{{- fail (printf "%s: %s.type %q is not one Azure takes (MySql, SQLServer, SQLAzure, Custom, NotificationHub, ServiceBus, EventHub, ApiHub, DocDb, RedisCache or PostgreSQL)" $root.Chart.Name $p (toString $c.type)) -}}
{{- end -}}
{{- $ref := include "aso.secretRef" (list $root (printf "%s.value" $p) $c.value) -}}
{{- if not $ref }}{{ fail (printf "%s: %s.value is required: a reference to a Kubernetes Secret, {secretName, key} (docs/secrets.md)" $root.Chart.Name $p) }}{{ end -}}
{{- $_ := set $byName $cn (dict "name" $cn "type" $c.type "valueFromSecret" (fromJson $ref)) -}}
{{- end -}}
{{- $items := list -}}
{{- range $k := keys $byName | sortAlpha }}{{ $items = append $items (index $byName $k) }}{{ end }}
---
apiVersion: web.azure.hfl.dev/v20260715
kind: SitesConfigConnectionStrings
metadata:
  {{- include "aso.metadata" (dict "root" .root "name" .name "id" .id "abandon" .abandon) | nindent 2 }}
spec:
  owner:
    {{- toYaml .owner | nindent 4 }}
  connectionStrings:
    {{- toYaml $items | nindent 4 }}
{{- end -}}
