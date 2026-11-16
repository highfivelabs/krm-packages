{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The eventbridge chart's own helpers: the bus's and role's names, the
values a chart cannot look up (account_id, region), the ARNs it builds
where ACK takes no reference, and IAM policy statements built as
Terraform's aws_iam_policy_document builds them (the step-functions
chart's helper, renamed).
*/}}

{{/*
eb.names: the AWS names of the bus and the role, as main.tf and iam.tf
derive them, with the namePrefix, or names.<key> in full. An existing bus
(create_bus: false) keeps bus_name as given. Returns JSON {bus, role}.
*/}}
{{- define "eb.names" -}}
{{- $v := .Values -}}
{{- $bus := $v.bus_name -}}
{{- if and $v.create $v.create_bus -}}
{{- if and (eq $v.bus_name "default") (not (index ($v.names | default dict) "bus")) -}}
{{- fail "eventbridge: create_bus: true with bus_name \"default\": every account and Region has a default bus, which cannot be created; set create_bus: false to use it, or name a new bus" -}}
{{- end -}}
{{- $bus = include "ack.resourceName" (list $ "bus" "event bus" $v.bus_name 256) -}}
{{- end -}}
{{- $role := "" -}}
{{- if and $v.create $v.create_role -}}
{{- $role = include "ack.resourceName" (list $ "role" "IAM role" ($v.role_name | default $v.bus_name) 64) -}}
{{- end -}}
{{- toJson (dict "bus" $bus "role" $role) -}}
{{- end -}}

{{/*
eb.key: a map key as main.tf turns it into a name, "_" to "-", with the
postfix when append is true.
  (list "orders_created" "-rule" $v.append_rule_postfix)
*/}}
{{- define "eb.key" -}}
{{- $k := index . 0 -}}
{{- if index . 2 -}}{{- printf "%s%s" (replace "_" "-" $k) (index . 1) -}}{{- else -}}{{- $k -}}{{- end -}}
{{- end -}}

{{/*
eb.need: a value a chart cannot look up (account_id, region), or fail.
  (list $ "account_id" "why")
*/}}
{{- define "eb.need" -}}
{{- $root := index . 0 -}}{{- $what := index . 1 -}}
{{- if not (index $root.Values $what) -}}
{{- fail (printf "eventbridge: %s is required for %s: the chart builds an ARN ACK takes no reference for, and cannot look the %s up (values.yaml, account_id)" $what (index . 2) (ternary "account" "Region" (eq $what "account_id"))) -}}
{{- end -}}
{{- end -}}

{{/*
eb.busArn: the bus's ARN, arn:<partition>:events:<region>:<account>:event-bus/<name>.
  (list $ "why")
*/}}
{{- define "eb.busArn" -}}
{{- $root := index . 0 -}}{{- $why := index . 1 -}}
{{- include "eb.need" (list $root "account_id" $why) -}}
{{- include "eb.need" (list $root "region" $why) -}}
{{- $n := include "eb.names" $root | fromJson -}}
{{- printf "arn:%s:events:%s:%s:event-bus/%s" $root.Values.partition $root.Values.region $root.Values.account_id $n.bus -}}
{{- end -}}

{{/*
eb.roleArn: the module's role's ARN (iam.tf, aws_iam_role.eventbridge).
  (list $ "why")
*/}}
{{- define "eb.roleArn" -}}
{{- $root := index . 0 -}}{{- $why := index . 1 -}}
{{- $v := $root.Values -}}
{{- if not (and $v.create $v.create_role) -}}
{{- fail (printf "eventbridge: %s needs the module's IAM role, and create_role is false: give the role's ARN" $why) -}}
{{- end -}}
{{- include "eb.need" (list $root "account_id" $why) -}}
{{- $n := include "eb.names" $root | fromJson -}}
{{- printf "arn:%s:iam::%s:role%s%s" $v.partition $v.account_id ($v.role_path | default "/") $n.role -}}
{{- end -}}

{{/*
eb.statement: one IAM policy statement as aws_iam_policy_document writes
it, from a dict in the module's shape. Returns JSON.
*/}}
{{- define "eb.statement" -}}
{{- $s := . -}}
{{- $o := dict "Effect" ($s.effect | default "Allow") -}}
{{- with $s.sid }}{{ $_ := set $o "Sid" . }}{{ end -}}
{{- with $s.actions }}{{ $_ := set $o "Action" . }}{{ end -}}
{{- with $s.not_actions }}{{ $_ := set $o "NotAction" . }}{{ end -}}
{{- with $s.resources }}{{ $_ := set $o "Resource" . }}{{ end -}}
{{- with $s.not_resources }}{{ $_ := set $o "NotResource" . }}{{ end -}}
{{- range $f := list (list "principals" "Principal") (list "not_principals" "NotPrincipal") -}}
{{- $ps := index $s (index $f 0) | default list -}}
{{- if $ps -}}
{{- $m := dict -}}
{{- range $p := $ps -}}
{{- $_ := set $m $p.type (concat (get $m $p.type | default list) $p.identifiers) -}}
{{- end -}}
{{- $_ := set $o (index $f 1) $m -}}
{{- end -}}
{{- end -}}
{{- $conds := $s.condition | default list -}}
{{- if $conds -}}
{{- $c := dict -}}
{{- range $x := $conds -}}
{{- $t := get $c $x.test | default dict -}}
{{- $vals := list -}}
{{- range $x.values }}{{ $vals = append $vals (toString .) }}{{ end -}}
{{- $_ := set $t $x.variable (concat (get $t $x.variable | default list) $vals | uniq) -}}
{{- $_ := set $c $x.test $t -}}
{{- end -}}
{{- $_ := set $o "Condition" $c -}}
{{- end -}}
{{- toJson $o -}}
{{- end -}}

{{/*
eb.secretRef: a secret value given as a reference to a Kubernetes Secret,
{secretName, key} (docs/secrets.md), as ACK's SecretKeyReference (YAML,
{name, key}). A string is refused: the value would be written into the
render.
  (list $ "connections.x.auth_parameters.basic.password" $value)
*/}}
{{- define "eb.secretRef" -}}
{{- $root := index . 0 -}}{{- $what := index . 1 -}}{{- $ref := index . 2 -}}
{{- if not (and (kindIs "map" $ref) $ref.secretName $ref.key) -}}
{{- fail (printf "eventbridge: %s is not supported as a value: give a reference to a Kubernetes Secret, {secretName, key} (docs/secrets.md); a value in values would be written into the render" $what) -}}
{{- end -}}
name: {{ $ref.secretName | quote }}
key: {{ $ref.key | quote }}
{{- end -}}

{{/*
eb.ecsNetwork: an ECS network configuration (subnets, security_groups,
assign_public_ip) as ACK's networkConfiguration.awsVPCConfiguration
(YAML). Terraform's provider sends assign_public_ip true as ENABLED.
*/}}
{{- define "eb.ecsNetwork" -}}
awsVPCConfiguration:
{{- with .subnets }}
  subnets: {{ toJson . }}
{{- end }}
{{- with .security_groups }}
  securityGroups: {{ toJson . }}
{{- end }}
{{- if not (kindIs "invalid" .assign_public_ip) }}
  assignPublicIP: {{ ternary "ENABLED" "DISABLED" (eq (toString .assign_public_ip) "true") | quote }}
{{- end }}
{{- end -}}

{{/*
eb.batching: a pipe source's batch_size and maximum_batching_window_in_seconds (YAML).
*/}}
{{- define "eb.batching" -}}
{{- if not (kindIs "invalid" .batch_size) }}
batchSize: {{ int64 .batch_size }}
{{- end }}
{{- if not (kindIs "invalid" .maximum_batching_window_in_seconds) }}
maximumBatchingWindowInSeconds: {{ int64 .maximum_batching_window_in_seconds }}
{{- end }}
{{- end -}}

{{/*
eb.stream: a DynamoDB or Kinesis stream source's parameters (YAML), without
Kinesis' starting_position_timestamp.
*/}}
{{- define "eb.stream" -}}
{{- include "eb.batching" . }}
{{- range $f := list (list "maximum_record_age_in_seconds" "maximumRecordAgeInSeconds") (list "maximum_retry_attempts" "maximumRetryAttempts") (list "parallelization_factor" "parallelizationFactor") }}
{{- $x := index $ (index $f 0) }}
{{- if and (hasKey $ (index $f 0)) (not (kindIs "invalid" $x)) }}
{{ index $f 1 }}: {{ int64 $x }}
{{- end }}
{{- end }}
{{- with .on_partial_batch_item_failure }}
onPartialBatchItemFailure: {{ . | quote }}
{{- end }}
{{- with .starting_position }}
startingPosition: {{ . | quote }}
{{- end }}
{{- with .dead_letter_config }}
deadLetterConfig:
  arn: {{ .arn | quote }}
{{- end }}
{{- end -}}

{{/*
eb.http: HTTP parameters (path_parameter_values, header_parameters,
query_string_parameters) as ACK's httpParameters (YAML).
*/}}
{{- define "eb.http" -}}
{{- with .path_parameter_values }}
pathParameterValues: {{ toJson . }}
{{- end }}
{{- with .header_parameters }}
headerParameters: {{ toJson . }}
{{- end }}
{{- with .query_string_parameters }}
queryStringParameters: {{ toJson . }}
{{- end }}
{{- end -}}

{{/*
eb.connParams: a connection's body, header and query_string parameters as
ack-extensions' bodyParameters, headerParameters, queryStringParameters
(YAML).
*/}}
{{- define "eb.connParams" -}}
{{- range $f := list (list "body" "bodyParameters") (list "header" "headerParameters") (list "query_string" "queryStringParameters") }}
{{- with get $ (index $f 0) }}
{{ index $f 1 }}:
{{- range . }}
  - key: {{ .key | quote }}
    value: {{ .value | toString | quote }}
{{- if not (kindIs "invalid" .is_value_secret) }}
    isValueSecret: {{ .is_value_secret }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}
