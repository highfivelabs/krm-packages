{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

alb.actions: a listener's default actions or a rule's actions, as ACK's
Action list (JSON), from one module action object (a listener, or one entry
of a rule's actions), in the order main.tf's dynamic blocks give them:
authenticate-cognito, authenticate-oidc, fixed-response, forward, weighted
forward, redirect. A forward to target_group_key references this chart's
TargetGroup (targetGroupRef); target_group_arn is taken as given.
jwt_validation has no ACK field (elbv2-controller v1.10.0) and is refused.
  {{ include "alb.actions" (dict "root" $ "a" $listener "where" "listeners.x") | fromJsonArray }}
*/}}
{{- define "alb.actions" -}}
{{- $root := .root -}}{{- $a := .a -}}{{- $where := .where -}}
{{- $order := $a.order -}}
{{- $out := list -}}
{{- $tgs := $root.Values.target_groups | default dict -}}
{{- if $a.jwt_validation -}}
{{- fail (printf "alb: %s: jwt_validation is not supported: ACK's Action (elbv2-controller v1.10.0) has no JWT validation config (docs/aws.md)" $where) -}}
{{- end -}}
{{- with $a.authenticate_cognito -}}
{{- $c := dict -}}
{{- with .authentication_request_extra_params }}{{ $_ := set $c "authenticationRequestExtraParams" . }}{{ end -}}
{{- with .on_unauthenticated_request }}{{ $_ := set $c "onUnauthenticatedRequest" . }}{{ end -}}
{{- with .scope }}{{ $_ := set $c "scope" . }}{{ end -}}
{{- with .session_cookie_name }}{{ $_ := set $c "sessionCookieName" . }}{{ end -}}
{{- if not (kindIs "invalid" .session_timeout) }}{{ $_ := set $c "sessionTimeout" (int64 .session_timeout) }}{{ end -}}
{{- with .user_pool_arn }}{{ $_ := set $c "userPoolARN" . }}{{ end -}}
{{- with .user_pool_client_id }}{{ $_ := set $c "userPoolClientID" . }}{{ end -}}
{{- with .user_pool_domain }}{{ $_ := set $c "userPoolDomain" . }}{{ end -}}
{{- $out = append $out (dict "type" "authenticate-cognito" "authenticateCognitoConfig" $c) -}}
{{- end -}}
{{- with $a.authenticate_oidc -}}
{{- $c := dict "authorizationEndpoint" .authorization_endpoint "clientID" .client_id "clientSecret" .client_secret "issuer" .issuer "tokenEndpoint" .token_endpoint "userInfoEndpoint" .user_info_endpoint -}}
{{- with .authentication_request_extra_params }}{{ $_ := set $c "authenticationRequestExtraParams" . }}{{ end -}}
{{- with .on_unauthenticated_request }}{{ $_ := set $c "onUnauthenticatedRequest" . }}{{ end -}}
{{- with .scope }}{{ $_ := set $c "scope" . }}{{ end -}}
{{- with .session_cookie_name }}{{ $_ := set $c "sessionCookieName" . }}{{ end -}}
{{- if not (kindIs "invalid" .session_timeout) }}{{ $_ := set $c "sessionTimeout" (int64 .session_timeout) }}{{ end -}}
{{- $out = append $out (dict "type" "authenticate-oidc" "authenticateOIDCConfig" $c) -}}
{{- end -}}
{{- with $a.fixed_response -}}
{{- $c := dict "contentType" .content_type -}}
{{- with .message_body }}{{ $_ := set $c "messageBody" . }}{{ end -}}
{{- with .status_code }}{{ $_ := set $c "statusCode" (toString .) }}{{ end -}}
{{- $out = append $out (dict "type" "fixed-response" "fixedResponseConfig" $c) -}}
{{- end -}}
{{- with $a.forward -}}
{{- $act := dict "type" "forward" -}}
{{- if .target_group_key -}}
{{- if not (hasKey $tgs .target_group_key) -}}
{{- fail (printf "alb: %s: forward.target_group_key %q is not a key of target_groups" $where .target_group_key) -}}
{{- end -}}
{{- $_ := set $act "targetGroupRef" (dict "from" (dict "name" (include "ack.name" (list $root.Release.Name "tg" .target_group_key)))) -}}
{{- else if .target_group_arn -}}
{{- $_ := set $act "targetGroupARN" .target_group_arn -}}
{{- else -}}
{{- fail (printf "alb: %s: forward needs target_group_key or target_group_arn" $where) -}}
{{- end -}}
{{- $out = append $out $act -}}
{{- end -}}
{{- with $a.weighted_forward -}}
{{- $groups := list -}}
{{- range $g := (.target_groups | default list) -}}
{{- $e := dict -}}
{{- if $g.target_group_key -}}
{{- if not (hasKey $tgs $g.target_group_key) -}}
{{- fail (printf "alb: %s: weighted_forward target_group_key %q is not a key of target_groups" $where $g.target_group_key) -}}
{{- end -}}
{{- $_ := set $e "targetGroupRef" (dict "from" (dict "name" (include "ack.name" (list $root.Release.Name "tg" $g.target_group_key)))) -}}
{{- else -}}
{{- $_ := set $e "targetGroupARN" $g.target_group_arn -}}
{{- end -}}
{{- if not (kindIs "invalid" $g.weight) }}{{ $_ := set $e "weight" (int64 $g.weight) }}{{ end -}}
{{- $groups = append $groups $e -}}
{{- end -}}
{{- $fc := dict "targetGroups" $groups -}}
{{- with .stickiness -}}
{{- $s := dict -}}
{{- if not (kindIs "invalid" .duration) }}{{ $_ := set $s "durationSeconds" (int64 .duration) }}{{ end -}}
{{- if not (kindIs "invalid" .enabled) }}{{ $_ := set $s "enabled" .enabled }}{{ end -}}
{{- $_ := set $fc "targetGroupStickinessConfig" $s -}}
{{- end -}}
{{- $out = append $out (dict "type" "forward" "forwardConfig" $fc) -}}
{{- end -}}
{{- with $a.redirect -}}
{{- $c := dict "statusCode" .status_code -}}
{{- range $f := list "host" "path" "port" "protocol" "query" -}}
{{- if not (kindIs "invalid" (index $.a.redirect $f)) }}{{ $_ := set $c $f (toString (index $.a.redirect $f)) }}{{ end -}}
{{- end -}}
{{- $out = append $out (dict "type" "redirect" "redirectConfig" $c) -}}
{{- end -}}
{{- $final := list -}}
{{- range $act := $out -}}
{{- if not (kindIs "invalid" $order) }}{{ $_ := set $act "order" (int64 $order) }}{{ end -}}
{{- $final = append $final $act -}}
{{- end -}}
{{- toJson $final -}}
{{- end -}}
