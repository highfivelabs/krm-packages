{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The elasticache chart's own helpers.
*/}}

{{/*
elasticache.tags: the module's local.tags, var.tags merged with
{terraform-aws-modules = "elasticache"}, then any further maps, as JSON.
  (list $ $more...)
*/}}
{{- define "elasticache.tags" -}}
{{- $root := index . 0 -}}
{{- include "ack.tags" (concat (list $root $root.Values.tags (dict "terraform-aws-modules" "elasticache")) (rest .)) -}}
{{- end -}}

{{/*
elasticache.base: coalesce(cluster_id, replication_group_id), lower-cased
as ElastiCache stores them, or "". The module derives the subnet group's,
security group's, parameter group's and log groups' names from it.
*/}}
{{- define "elasticache.base" -}}
{{- $v := .Values -}}
{{- coalesce $v.cluster_id $v.replication_group_id | default "" | lower -}}
{{- end -}}

{{/*
elasticache.port: coalesce(var.port, 11211 for Memcached, else 6379).
*/}}
{{- define "elasticache.port" -}}
{{- $v := .Values -}}
{{- if not (kindIs "invalid" $v.port) }}{{ int64 $v.port }}{{ else if eq (toString $v.engine) "memcached" }}11211{{ else }}6379{{ end -}}
{{- end -}}

{{/*
elasticache.logGroups: the log groups the module makes
(aws_cloudwatch_log_group.this), as JSON {key: name}: one per
log_delivery_configuration entry with destination_type cloudwatch-logs and
create_cloudwatch_log_group not false, unless the engine is Memcached.
Named /aws/elasticache/<cloudwatch_log_group_name, else cluster_id, else
replication_group_id>, with the namePrefix on the last part.
*/}}
{{- define "elasticache.logGroups" -}}
{{- $root := . -}}
{{- $v := .Values -}}
{{- $out := dict -}}
{{- if and $v.create (ne (toString $v.engine) "memcached") -}}
{{- range $key, $l := ($v.log_delivery_configuration | default dict) -}}
{{- $l = $l | default dict -}}
{{- if and (ne (toJson $l.create_cloudwatch_log_group) "false") (eq (toString $l.destination_type) "cloudwatch-logs") -}}
{{- $n := $l.cloudwatch_log_group_name | default (include "elasticache.base" $root) -}}
{{- if $n }}{{ $n = include "ack.cloudName" (list $root "log group" $n 495) }}{{ end -}}
{{- $_ := set $out $key (printf "/aws/elasticache/%s" $n) -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- toJson $out -}}
{{- end -}}

{{/*
elasticache.logDelivery: spec.logDeliveryConfigurations from
log_delivery_configuration, as the module's dynamic block writes them: a
CloudWatch Logs destination the chart makes is its log group's name.
Renders nothing for Memcached or no entries.
*/}}
{{- define "elasticache.logDelivery" -}}
{{- $root := . -}}
{{- $v := .Values -}}
{{- $groups := include "elasticache.logGroups" $root | fromJson -}}
{{- $lines := list -}}
{{- if ne (toString $v.engine) "memcached" -}}
{{- range $key := keys ($v.log_delivery_configuration | default dict) | sortAlpha -}}
{{- $l := index $v.log_delivery_configuration $key | default dict -}}
{{- $dest := $l.destination -}}
{{- if hasKey $groups $key }}{{ $dest = index $groups $key }}{{ end -}}
{{- if not $l.destination_type -}}
{{- fail (printf "elasticache: log_delivery_configuration.%s needs destination_type (cloudwatch-logs or kinesis-firehose)" $key) -}}
{{- end -}}
{{- if not $dest -}}
{{- fail (printf "elasticache: log_delivery_configuration.%s needs destination: the log group's or delivery stream's name" $key) -}}
{{- end -}}
{{- $details := dict -}}
{{- if eq (toString $l.destination_type) "cloudwatch-logs" -}}
{{- $details = dict "cloudWatchLogsDetails" (dict "logGroup" $dest) -}}
{{- else -}}
{{- $details = dict "kinesisFirehoseDetails" (dict "deliveryStream" $dest) -}}
{{- end -}}
{{- $e := dict "destinationType" $l.destination_type "destinationDetails" $details "enabled" true "logType" ($l.log_type | default $key) -}}
{{- with $l.log_format }}{{ $_ := set $e "logFormat" . }}{{ end -}}
{{- $lines = append $lines (printf "- %s" (toJson $e)) -}}
{{- end -}}
{{- end -}}
{{- join "\n" $lines -}}
{{- end -}}

{{/*
elasticache.secretRef: secret material (the replication group's AUTH
token) as a reference to a Kubernetes Secret in the component's namespace,
never as a value (docs/secrets.md): {secretName, key} becomes ACK's
SecretKeyReference {name, key}, as JSON, or "" when the value is null. The
Secret is expected to come from External Secrets Operator. A string (the
secret itself) or any other shape stops the render.
  {{ include "elasticache.secretRef" (list $ "auth_token" .Values.auth_token) }}
*/}}
{{- define "elasticache.secretRef" -}}
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
