{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

ECS chart helpers: IAM policy document builder and security group rule
translation.
*/ -}}

{{- /*
ecs.iamStatement: converts a Terraform-style statement map into an IAM
policy statement dict.
*/ -}}
{{- define "ecs.iamStatement" -}}
{{- $s := . -}}
{{- $out := dict "Effect" ($s.effect | default "Allow") "Action" $s.actions "Resource" $s.resources -}}
{{- with $s.sid }}{{ $_ := set $out "Sid" . }}{{ end -}}
{{- with $s.not_actions }}{{ $_ := set $out "NotAction" . }}{{ $_ := unset $out "Action" }}{{ end -}}
{{- with $s.not_resources }}{{ $_ := set $out "NotResource" . }}{{ $_ := unset $out "Resource" }}{{ end -}}
{{- with $s.condition }}
{{- $cond := dict -}}
{{- range $c := . }}
{{- $op := index $cond $c.test | default dict -}}
{{- $_ := set $op $c.variable $c.values -}}
{{- $_ := set $cond $c.test $op -}}
{{- end }}
{{- $_ := set $out "Condition" $cond -}}
{{- end }}
{{- toJson $out -}}
{{- end -}}

{{- /*
ecs.assumeRolePolicy: builds an AssumeRolePolicyDocument JSON string for an
AWS service principal.
*/ -}}
{{- define "ecs.assumeRolePolicy" -}}
{{- $svc := index . 0 -}}
{{- toJson (dict "Version" "2012-10-17" "Statement" (list (dict "Effect" "Allow" "Principal" (dict "Service" $svc) "Action" "sts:AssumeRole"))) -}}
{{- end -}}

{{- /*
ecs.sgRule: converts a terraform-aws-modules security_group_*_rules entry into
an ACK ec2 SecurityGroup IPPermission dict.
*/ -}}
{{- define "ecs.sgRule" -}}
{{- $r := . -}}
{{- $proto := $r.ip_protocol | default "tcp" -}}
{{- $out := dict "ipProtocol" (printf "%v" $proto) -}}
{{- if not (kindIs "invalid" $r.from_port) }}{{ $_ := set $out "fromPort" (int64 $r.from_port) }}{{ end -}}
{{- if not (kindIs "invalid" $r.to_port) }}{{ $_ := set $out "toPort" (int64 $r.to_port) }}{{ end -}}
{{- with $r.cidr_ipv4 }}
{{- $range := dict "cidrIP" . -}}
{{- with $r.description }}{{ $_ := set $range "description" . }}{{ end -}}
{{- $_ := set $out "ipRanges" (list $range) -}}
{{- end }}
{{- with $r.cidr_ipv6 }}
{{- $range := dict "cidrIPv6" . -}}
{{- with $r.description }}{{ $_ := set $range "description" . }}{{ end -}}
{{- $_ := set $out "ipv6Ranges" (list $range) -}}
{{- end }}
{{- with $r.prefix_list_id }}
{{- $pl := dict "prefixListID" . -}}
{{- with $r.description }}{{ $_ := set $pl "description" . }}{{ end -}}
{{- $_ := set $out "prefixListIDs" (list $pl) -}}
{{- end }}
{{- with $r.referenced_security_group_id }}
{{- $ug := dict "groupID" . -}}
{{- with $r.description }}{{ $_ := set $ug "description" . }}{{ end -}}
{{- $_ := set $out "userIDGroupPairs" (list $ug) -}}
{{- end }}
{{- toJson $out -}}
{{- end -}}
