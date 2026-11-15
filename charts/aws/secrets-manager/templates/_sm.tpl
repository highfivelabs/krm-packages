{{/*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

The secrets-manager chart's own helpers: the secret's name, and the
resource policy built as Terraform's aws_iam_policy_document builds it (the
helpers are the sqs chart's, renamed).
*/}}

{{/*
sm.name: the secret's AWS name, var.name with the namePrefix, or
names.secret in full ("" when neither is set).
*/}}
{{- define "sm.name" -}}
{{- $v := .Values -}}
{{- if index ($v.names | default dict) "secret" -}}
{{- include "ack.resourceName" (list $ "secret" "secret" "" 512) -}}
{{- else if $v.name -}}
{{- include "ack.cloudName" (list $ "secret" $v.name 512) -}}
{{- end -}}
{{- end -}}

{{/*
sm.statement: one IAM policy statement as aws_iam_policy_document
writes it, from a dict in the module's shape. Returns JSON.
*/}}
{{- define "sm.statement" -}}
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
sm.statements: the statements of one JSON policy document, as a JSON
list (Statement may be one object or a list).
  (list "source_queue_policy_documents[0]" $json)
*/}}
{{- define "sm.statements" -}}
{{- $what := index . 0 -}}{{- $json := index . 1 -}}
{{- $doc := $json | fromJson -}}
{{- if or (hasKey $doc "Error") (not (hasKey $doc "Statement")) -}}
{{- fail (printf "secrets-manager: %s is not a JSON policy document with a Statement" $what) -}}
{{- end -}}
{{- $st := $doc.Statement -}}
{{- if kindIs "map" $st }}{{ $st = list $st }}{{ end -}}
{{- toJson (dict "s" $st) -}}
{{- end -}}

{{/*
sm.merge: statements merged into a document's, as aws_iam_policy_document
does: a statement with a Sid replaces the one with the same Sid in place,
any other is appended. Returns JSON {s: [...]}.
*/}}
{{- define "sm.merge" -}}
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
sm.policy: the resource policy as main.tf's data.aws_iam_policy_document
builds it: the source documents, the statements (a map, in key order, each
Sid defaulting to its key), then the override documents, merged by Sid.
The module gives a statement no default resource. Returns the JSON, or ""
when it has no statement.
  (list $sources $statements $overrides)
*/}}
{{- define "sm.policy" -}}
{{- $sources := index . 0 | default list -}}{{- $m := index . 1 | default dict -}}{{- $overrides := index . 2 | default list -}}
{{- $doc := list -}}
{{- $sids := dict -}}
{{- range $i, $j := $sources -}}
{{- $st := (include "sm.statements" (list (printf "source_policy_documents[%d]" $i) $j) | fromJson).s -}}
{{- range $s := $st -}}
{{- with get $s "Sid" -}}
{{- if hasKey $sids . -}}
{{- fail (printf "secrets-manager: source_policy_documents: duplicate Sid (%s); the source documents' Sids must be unique" .) -}}
{{- end -}}
{{- $_ := set $sids . true -}}
{{- end -}}
{{- end -}}
{{- $doc = (include "sm.merge" (list $doc $st) | fromJson).s -}}
{{- end -}}
{{- $own := list -}}
{{- range $k := keys $m | sortAlpha -}}
{{- $s := deepCopy (index $m $k | default dict) -}}
{{- $_ := set $s "sid" ($s.sid | default $k) -}}
{{- $own = append $own (include "sm.statement" $s | fromJson) -}}
{{- end -}}
{{- $doc = (include "sm.merge" (list $doc $own) | fromJson).s -}}
{{- range $i, $j := $overrides -}}
{{- $st := (include "sm.statements" (list (printf "override_policy_documents[%d]" $i) $j) | fromJson).s -}}
{{- $doc = (include "sm.merge" (list $doc $st) | fromJson).s -}}
{{- end -}}
{{- if $doc -}}
{{- toJson (dict "Version" "2012-10-17" "Statement" $doc) -}}
{{- end -}}
{{- end -}}
