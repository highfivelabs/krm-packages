{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The eks chart's own helpers: IAM policy documents built as Terraform's
aws_iam_policy_document builds them (the eks-pod-identity chart's
helpers, renamed), the cluster KMS key's policy (terraform-aws-kms
v4.0.0, as the module calls it), an IAM role, and security group rules in
the module's aws_security_group_rule shape.
*/}}

{{/*
eks.statement: one IAM policy statement as aws_iam_policy_document
writes it, from a dict in the module's shape. Returns JSON.
*/}}
{{- define "eks.statement" -}}
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
eks.statements: the statements of one JSON policy document, as a JSON
list (Statement may be one object or a list).
  (list "kms_key_source_policy_documents[0]" $json)
*/}}
{{- define "eks.statements" -}}
{{- $what := index . 0 -}}{{- $json := index . 1 -}}
{{- $doc := $json | fromJson -}}
{{- if or (hasKey $doc "Error") (not (hasKey $doc "Statement")) -}}
{{- fail (printf "eks: %s is not a JSON policy document with a Statement" $what) -}}
{{- end -}}
{{- $st := $doc.Statement -}}
{{- if kindIs "map" $st }}{{ $st = list $st }}{{ end -}}
{{- toJson (dict "s" $st) -}}
{{- end -}}

{{/*
eks.merge: statements merged into a document's, as aws_iam_policy_document
does: a statement with a Sid replaces the one with the same Sid in place,
any other is appended. Returns JSON {s: [...]}.
*/}}
{{- define "eks.merge" -}}
{{- $out := index . 0 -}}
{{- range $n := index . 1 -}}
{{- $sid := get $n "Sid" | default "" -}}
{{- $done := false -}}
{{- if $sid -}}
{{- $next := list -}}
{{- range $e := $out -}}
{{- if and (not $done) (eq (get $e "Sid" | default "") $sid) -}}
{{- $next = append $next $n -}}{{- $done = true -}}
{{- else -}}
{{- $next = append $next $e -}}
{{- end -}}
{{- end -}}
{{- $out = $next -}}
{{- end -}}
{{- if not $done }}{{ $out = append $out $n }}{{ end -}}
{{- end -}}
{{- toJson (dict "s" $out) -}}
{{- end -}}

{{/*
eks.document: a policy document as aws_iam_policy_document builds it: the
source documents, the statements (a list, each in the module's shape),
then the override documents, merged by Sid. Returns the JSON, or "" when
it has no statement.
  (list "what" $sources $statements $overrides)
*/}}
{{- define "eks.document" -}}
{{- $what := index . 0 -}}
{{- $sources := index . 1 | default list -}}{{- $stmts := index . 2 | default list -}}{{- $overrides := index . 3 | default list -}}
{{- $doc := list -}}
{{- range $i, $j := $sources -}}
{{- $st := (include "eks.statements" (list (printf "%s source_policy_documents[%d]" $what $i) $j) | fromJson).s -}}
{{- $doc = (include "eks.merge" (list $doc $st) | fromJson).s -}}
{{- end -}}
{{- $own := list -}}
{{- range $s := $stmts -}}
{{- $own = append $own (include "eks.statement" $s | fromJson) -}}
{{- end -}}
{{- $doc = (include "eks.merge" (list $doc $own) | fromJson).s -}}
{{- range $i, $j := $overrides -}}
{{- $st := (include "eks.statements" (list (printf "%s override_policy_documents[%d]" $what $i) $j) | fromJson).s -}}
{{- $doc = (include "eks.merge" (list $doc $st) | fromJson).s -}}
{{- end -}}
{{- if $doc -}}
{{- toJson (dict "Version" "2012-10-17" "Statement" $doc) -}}
{{- end -}}
{{- end -}}

{{/*
eks.kmsPolicy: the cluster key's policy, as terraform-aws-kms v4.0.0's
aws_iam_policy_document builds it with the arguments main.tf ("KMS Key")
passes: the Default statement, the key owners, administrators, users (the
cluster role first) and service users, merged with the source and
override documents.
  (list $ $roleARN)
*/}}
{{- define "eks.kmsPolicy" -}}
{{- $root := index . 0 -}}{{- $roleARN := index . 1 -}}
{{- $v := $root.Values -}}
{{- $st := list -}}
{{- if $v.kms_key_enable_default_policy -}}
{{- $st = append $st (dict "sid" "Default" "actions" (list "kms:*") "resources" (list "*") "principals" (list (dict "type" "AWS" "identifiers" (list (printf "arn:%s:iam::%s:root" $v.partition $v.account_id))))) -}}
{{- end -}}
{{- $users := concat (list $roleARN) ($v.kms_key_users | default list) -}}
{{- $groups := list
  (list "KeyOwner" $v.kms_key_owners (list "kms:*") list)
  (list "KeyAdministration" $v.kms_key_administrators (list "kms:Create*" "kms:Describe*" "kms:Enable*" "kms:List*" "kms:Put*" "kms:Update*" "kms:Revoke*" "kms:Disable*" "kms:Get*" "kms:Delete*" "kms:TagResource" "kms:UntagResource" "kms:ScheduleKeyDeletion" "kms:CancelKeyDeletion" "kms:ReplicateKey" "kms:ImportKeyMaterial") list)
  (list "KeyUsage" $users (list "kms:Encrypt" "kms:Decrypt" "kms:ReEncrypt*" "kms:GenerateDataKey*" "kms:DescribeKey") list)
  (list "KeyServiceUsage" $v.kms_key_service_users (list "kms:CreateGrant" "kms:ListGrants" "kms:RevokeGrant") (list (dict "test" "Bool" "variable" "kms:GrantIsForAWSResource" "values" (list "true")))) -}}
{{- range $g := $groups -}}
{{- if index $g 1 -}}
{{- $st = append $st (dict "sid" (index $g 0) "actions" (index $g 2) "resources" (list "*") "principals" (list (dict "type" "AWS" "identifiers" (index $g 1))) "condition" (index $g 3)) -}}
{{- end -}}
{{- end -}}
{{- include "eks.document" (list "kms key policy" $v.kms_key_source_policy_documents $st $v.kms_key_override_policy_documents) -}}
{{- end -}}

{{/*
eks.role: an IAM role (aws_iam_role, iam/Role) with its policy
attachments (aws_iam_role_policy_attachment, as policy ARNs: ACK's Role
takes ARNs or references, not both) and an inline policy
(aws_iam_role_policy). force_detach_policies is ACK's behaviour: it
detaches a role's policies before deleting it.
  dict: root, k8s (object name), id, name, path, description, boundary,
  trust (JSON), arns (list), inline (dict name: JSON), tags (dict)
*/}}
{{- define "eks.role" }}
---
apiVersion: iam.services.k8s.aws/v1alpha1
kind: Role
metadata:
{{- include "ack.metadata" (dict "root" .root "name" .k8s "id" .id) | nindent 2 }}
spec:
  name: {{ .name | quote }}
  path: {{ .path | default "/" | quote }}
{{- with .description }}
  description: {{ . | quote }}
{{- end }}
{{- with .boundary }}
  permissionsBoundary: {{ . | quote }}
{{- end }}
  assumeRolePolicyDocument: {{ .trust | quote }}
{{- with .arns }}
  policies:
{{- range . }}
    - {{ . | quote }}
{{- end }}
{{- end }}
{{- with .inline }}
  inlinePolicies:
{{- range $k := keys . | sortAlpha }}
    {{ $k | quote }}: {{ index $.inline $k | quote }}
{{- end }}
{{- end }}
{{- with .tags }}
  tags:
{{- include "ack.tagList" (list .) | nindent 4 }}
{{- end }}
{{- end -}}

{{/*
eks.arn: an IAM role's or policy's ARN, as IAM builds it, for a role
or policy this render makes and attaches by ARN.
  (list $ "role" $path $name "what needs it")
*/}}
{{- define "eks.arn" -}}
{{- $root := index . 0 -}}{{- $type := index . 1 -}}{{- $path := index . 2 | default "/" -}}{{- $name := index . 3 -}}{{- $why := index . 4 -}}
{{- if not $root.Values.account_id -}}
{{- fail (printf "eks: account_id is required for %s: the chart builds the %s's ARN, which names the account, and cannot look the account up (values.yaml, account_id)" $why $type) -}}
{{- end -}}
{{- printf "arn:%s:iam::%s:%s%s%s" $root.Values.partition $root.Values.account_id $type $path $name -}}
{{- end -}}

{{/*
eks.rules: a security group's rules from the module's
aws_security_group_rule maps, as ACK's SecurityGroup ingressRules and
egressRules (YAML, indented for spec). Each rule's source: cidr_blocks,
ipv6_cidr_blocks, prefix_list_ids, self (a pair without a group, which
ACK fills with the group's own) and a source group, given as a pair from
the "source" dict {ref: <object name>} or {id: <group ID>} when the rule's
source flag (sourceFlag) is set, else source_security_group_id.
  dict: root, rules (dict), sourceFlag, source (dict), what
*/}}
{{- define "eks.rules" -}}
{{- $rules := .rules -}}
{{- range $dir := list "ingress" "egress" }}
{{- $keys := list }}
{{- range $k := keys $rules | sortAlpha }}{{ if eq ((index $rules $k).type | default "ingress") $dir }}{{ $keys = append $keys $k }}{{ end }}{{ end }}
{{- if $keys }}
{{ $dir }}Rules:
{{- range $k := $keys }}
{{- $r := index $rules $k }}
{{- $proto := $r.protocol | default "tcp" | toString }}
{{- $desc := dict }}
{{- with $r.description }}{{ $desc = dict "description" . }}{{ end }}
  - ipProtocol: {{ $proto | quote }}
{{- if ne $proto "-1" }}
    fromPort: {{ int64 $r.from_port }}
    toPort: {{ int64 $r.to_port }}
{{- end }}
{{- with $r.cidr_blocks }}
    ipRanges:
{{- range . }}
      - {{ toJson (merge (dict "cidrIP" .) $desc) }}
{{- end }}
{{- end }}
{{- with $r.ipv6_cidr_blocks }}
    ipv6Ranges:
{{- range . }}
      - {{ toJson (merge (dict "cidrIPv6" .) $desc) }}
{{- end }}
{{- end }}
{{- with $r.prefix_list_ids }}
    prefixListIDs:
{{- range . }}
      - {{ toJson (merge (dict "prefixListID" .) $desc) }}
{{- end }}
{{- end }}
{{- $pairs := list }}
{{- if $r.self }}{{ $pairs = append $pairs $desc }}{{ end }}
{{- if index $r $.sourceFlag }}
{{- if $.source.ref }}
{{- $pairs = append $pairs (merge (dict "groupRef" (dict "from" (dict "name" $.source.ref))) $desc) }}
{{- else if $.source.id }}
{{- $pairs = append $pairs (merge (dict "groupID" $.source.id) $desc) }}
{{- else }}
{{- fail (printf "eks: %s.%s: %s needs that security group, which this component does not make and whose ID is not given" $.what $k $.sourceFlag) }}
{{- end }}
{{- else if $r.source_security_group_id }}
{{- $pairs = append $pairs (merge (dict "groupID" $r.source_security_group_id) $desc) }}
{{- end }}
{{- if $pairs }}
    userIDGroupPairs:
{{- range $pairs }}
      - {{ toJson . }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}
