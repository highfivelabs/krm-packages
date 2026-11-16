{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The rds-aurora chart's own helpers.
*/}}

{{/*
aurora.on: "true" when the module creates anything (local.create: create
and putin_khuylo), else "".
*/}}
{{- define "aurora.on" -}}
{{- if and .Values.create .Values.putin_khuylo }}true{{ end -}}
{{- end -}}

{{/*
aurora.port: local.port, coalesce(var.port, 5432 for aurora-postgresql or
postgres, else 3306).
*/}}
{{- define "aurora.port" -}}
{{- $v := .Values -}}
{{- if $v.port }}{{ int64 $v.port }}{{ else if has (toString $v.engine) (list "aurora-postgresql" "postgres") }}5432{{ else }}3306{{ end -}}
{{- end -}}

{{/*
aurora.serverless: "true" for engine_mode serverless (local.is_serverless),
which makes no instances, endpoints or autoscaling.
*/}}
{{- define "aurora.serverless" -}}
{{- if eq (toString .Values.engine_mode) "serverless" }}true{{ end -}}
{{- end -}}

{{/*
aurora.clusterID: the cluster's identifier, var.name lower-cased as RDS
stores it, with the namePrefix (names.cluster in full).
*/}}
{{- define "aurora.clusterID" -}}
{{- if not .Values.name -}}
{{- fail "rds-aurora: name is required: the cluster's identifier, which the module also names its groups, role and log groups after" -}}
{{- end -}}
{{- include "ack.resourceName" (list . "cluster" "DB cluster identifier" (lower .Values.name) 63 "letter") -}}
{{- end -}}

{{/*
aurora.instances: the instances the module makes (aws_rds_cluster_instance,
none when serverless), as JSON {key: identifier}: the instance's
identifier with the namePrefix, else <cluster identifier>-<key>.
*/}}
{{- define "aurora.instances" -}}
{{- $root := . -}}
{{- $out := dict -}}
{{- if and (include "aurora.on" $root) (not (include "aurora.serverless" $root)) -}}
{{- $cid := include "aurora.clusterID" $root -}}
{{- range $key, $i := (.Values.instances | default dict) -}}
{{- $i = $i | default dict -}}
{{- $id := "" -}}
{{- if $i.identifier -}}
{{- $id = include "ack.cloudName" (list $root "DB instance identifier" (lower $i.identifier) 63 "letter") -}}
{{- else -}}
{{- $id = lower (printf "%s-%s" $cid $key) -}}
{{- if gt (len $id) 63 -}}
{{- fail (printf "rds-aurora: instance %s's identifier %q is %d characters, more than the 63 RDS allows: set instances.%s.identifier" $key $id (len $id) $key) -}}
{{- end -}}
{{- end -}}
{{- $_ := set $out $key $id -}}
{{- end -}}
{{- end -}}
{{- toJson $out -}}
{{- end -}}

{{/*
aurora.monitoringRole: "true" when the module makes the enhanced
monitoring role (local.create_monitoring_role): create_monitoring_role,
and the cluster or an instance has a monitoring interval above 0.
*/}}
{{- define "aurora.monitoringRole" -}}
{{- $v := .Values -}}
{{- $any := gt (int64 ($v.cluster_monitoring_interval | default 0)) 0 -}}
{{- range $_, $i := ($v.instances | default dict) -}}
{{- if gt (int64 (($i | default dict).monitoring_interval | default 0)) 0 }}{{ $any = true }}{{ end -}}
{{- end -}}
{{- if and (include "aurora.on" .) $v.create_monitoring_role $any }}true{{ end -}}
{{- end -}}

{{/*
aurora.secretRef: a reference to a Kubernetes Secret, {secretName, key}
(docs/secrets.md), as JSON {name, key} for ACK's SecretKeyReference; ""
when unset. A string (the secret itself) is refused.
  (list $ "master_password_wo" $value)
*/}}
{{- define "aurora.secretRef" -}}
{{- $root := index . 0 -}}{{- $path := index . 1 -}}{{- $v := index . 2 -}}
{{- if not (kindIs "invalid" $v) -}}
{{- if not (kindIs "map" $v) -}}
{{- fail (printf "%s: %s takes a reference to a Kubernetes Secret, {secretName, key}, never the secret's value (docs/secrets.md)" $root.Chart.Name $path) -}}
{{- end -}}
{{- $n := $v.secretName -}}{{- $k := $v.key -}}
{{- if or (omit $v "secretName" "key" | keys) (not (kindIs "string" $n)) (not (kindIs "string" $k)) -}}
{{- fail (printf "%s: %s is {secretName, key}: the name of a Kubernetes Secret in the component's namespace and the key that holds the value (docs/secrets.md)" $root.Chart.Name $path) -}}
{{- end -}}
{{- if or (gt (len $n) 253) (not (regexMatch "^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$" $n)) -}}
{{- fail (printf "%s: %s.secretName %q is not a Kubernetes Secret name (lowercase letters, digits, '-' and '.', at most 253 characters)" $root.Chart.Name $path $n) -}}
{{- end -}}
{{- if or (gt (len $k) 253) (not (regexMatch "^[-._a-zA-Z0-9]+$" $k)) -}}
{{- fail (printf "%s: %s.key %q is not a Secret key (letters, digits, '-', '_' and '.')" $root.Chart.Name $path $k) -}}
{{- end -}}
{{- toJson (dict "name" $n "key" $k) -}}
{{- end -}}
{{- end -}}

{{/*
aurora.applyImmediately: refuse apply_immediately: false (the module's
cluster value, or an instance's). ACK applies every change immediately.
  (list $ "apply_immediately" $value)
*/}}
{{- define "aurora.applyImmediately" -}}
{{- $root := index . 0 -}}{{- $path := index . 1 -}}{{- $v := index . 2 -}}
{{- if and (not (kindIs "invalid" $v)) (not $v) -}}
{{- fail (printf "rds-aurora: %s: false is not supported: ACK applies a cluster's and its instances' changes immediately, as true does; there is no waiting for the maintenance window (docs/aws.md)" $path) -}}
{{- end -}}
{{- end -}}
