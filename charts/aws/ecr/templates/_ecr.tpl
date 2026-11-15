{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The ecr chart's own helpers: the repository's name, and the repository
policy built as the module's data.aws_iam_policy_document builds it (the
statement helper is the sqs chart's, renamed).
*/}}

{{/*
ecr.name: the repository's AWS name, repository_name with the namePrefix,
or names.repository in full ("" when neither is set).
*/}}
{{- define "ecr.name" -}}
{{- $v := .Values -}}
{{- if index ($v.names | default dict) "repository" -}}
{{- include "ack.resourceName" (list $ "repository" "repository" "" 256) -}}
{{- else if $v.repository_name -}}
{{- include "ack.cloudName" (list $ "repository" $v.repository_name 256) -}}
{{- end -}}
{{- end -}}

{{/*
ecr.statement: one IAM policy statement as aws_iam_policy_document
writes it, from a dict in the module's shape. Returns JSON.
*/}}
{{- define "ecr.statement" -}}
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
{{- if and (eq (len $m) 1) (hasKey $m "*") (eq (toJson (get $m "*")) "[\"*\"]") -}}
{{- /* type "*" with identifiers ["*"]: aws_iam_policy_document writes "*" */ -}}
{{- $_ := set $o (index $f 1) "*" -}}
{{- else -}}
{{- $_ := set $o (index $f 1) $m -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- $conds := concat ($s.condition | default list) ($s.conditions | default list) -}}
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
ecr.policy: the repository policy as main.tf's
data.aws_iam_policy_document.repository builds it for a private
repository: PrivateReadOnly (the read and read/write ARNs, or the
account's root), PrivateLambdaReadOnly (when Lambda ARNs are given),
ReadWrite (when read/write ARNs are given), then repository_policy_statements
in key order (their sid as given: the module gives no default). Returns the
JSON.
*/}}
{{- define "ecr.policy" -}}
{{- $v := .Values -}}
{{- $readers := concat ($v.repository_read_access_arns | default list) ($v.repository_read_write_access_arns | default list) -}}
{{- if not $readers -}}
{{- if not $v.account_id -}}
{{- fail "ecr: account_id is required for the default repository policy: with no repository_read_access_arns or repository_read_write_access_arns its principal is the account's root, arn:<partition>:iam::<account_id>:root, which a chart cannot look up (values.yaml, account_id)" -}}
{{- end -}}
{{- $readers = list (printf "arn:%s:iam::%s:root" $v.partition $v.account_id) -}}
{{- end -}}
{{- $doc := list -}}
{{- $doc = append $doc (include "ecr.statement" (dict "sid" "PrivateReadOnly" "principals" (list (dict "type" "AWS" "identifiers" $readers)) "actions" (list "ecr:GetAuthorizationToken" "ecr:BatchCheckLayerAvailability" "ecr:BatchGetImage" "ecr:DescribeImageScanFindings" "ecr:DescribeImages" "ecr:DescribeRepositories" "ecr:GetDownloadUrlForLayer" "ecr:GetLifecyclePolicy" "ecr:GetLifecyclePolicyPreview" "ecr:GetRepositoryPolicy" "ecr:ListImages" "ecr:ListTagsForResource")) | fromJson) -}}
{{- with $v.repository_lambda_read_access_arns -}}
{{- $doc = append $doc (include "ecr.statement" (dict "sid" "PrivateLambdaReadOnly" "principals" (list (dict "type" "Service" "identifiers" (list "lambda.amazonaws.com"))) "actions" (list "ecr:BatchGetImage" "ecr:GetDownloadUrlForLayer") "conditions" (list (dict "test" "StringLike" "variable" "aws:sourceArn" "values" .))) | fromJson) -}}
{{- end -}}
{{- with $v.repository_read_write_access_arns -}}
{{- $doc = append $doc (include "ecr.statement" (dict "sid" "ReadWrite" "principals" (list (dict "type" "AWS" "identifiers" .)) "actions" (list "ecr:PutImage" "ecr:InitiateLayerUpload" "ecr:UploadLayerPart" "ecr:CompleteLayerUpload")) | fromJson) -}}
{{- end -}}
{{- $m := $v.repository_policy_statements | default dict -}}
{{- range $k := keys $m | sortAlpha -}}
{{- $doc = append $doc (include "ecr.statement" (index $m $k | default dict) | fromJson) -}}
{{- end -}}
{{- toJson (dict "Version" "2012-10-17" "Statement" $doc) -}}
{{- end -}}
