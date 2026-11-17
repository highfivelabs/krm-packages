{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

Helpers for s3-bucket: status normalization, filter normalization (single
predicate vs And conjunction), and the module's combined bucket policy
document.
*/ -}}

{{- /* s3.status: bool/string -> "Enabled" | "Disabled" */ -}}
{{- define "s3.status" -}}
{{- $v := . -}}
{{- $s := toString $v -}}
{{- if or (eq $s "true") (eq (lower $s) "enabled") -}}Enabled{{- else -}}Disabled{{- end -}}
{{- end -}}

{{- /* s3.tagList: map of k->v (or list of {key,value}) -> list of {key, value} sorted by key */ -}}
{{- define "s3.tagList" -}}
{{- $tags := . -}}
{{- $out := list -}}
{{- if kindIs "map" $tags -}}
{{- range $k := keys $tags | sortAlpha -}}
{{- $out = append $out (dict "key" $k "value" (toString (index $tags $k))) -}}
{{- end -}}
{{- else if kindIs "slice" $tags -}}
{{- $out = $tags -}}
{{- end -}}
{{- toJson $out -}}
{{- end -}}

{{- /*
s3.filter: converts a Terraform filter map (prefix, tag/tags,
object_size_greater_than, object_size_less_than, access_point) into ACK's
filter object (single predicate directly on filter, >1 under filter.and).
*/ -}}
{{- define "s3.filter" -}}
{{- $f := . | default dict -}}
{{- $tagMap := $f.tags | default $f.tag | default dict -}}
{{- $tags := include "s3.tagList" $tagMap | fromJsonArray -}}
{{- $count := len $tags -}}
{{- if $f.prefix }}{{ $count = add $count 1 }}{{ end -}}
{{- if $f.access_point }}{{ $count = add $count 1 }}{{ end -}}
{{- if not (kindIs "invalid" $f.object_size_greater_than) }}{{ $count = add $count 1 }}{{ end -}}
{{- if not (kindIs "invalid" $f.object_size_less_than) }}{{ $count = add $count 1 }}{{ end -}}
{{- $out := dict -}}
{{- if eq $count 1 -}}
{{- if $f.prefix }}{{ $_ := set $out "prefix" $f.prefix }}{{ end -}}
{{- if $f.access_point }}{{ $_ := set $out "accessPointARN" $f.access_point }}{{ end -}}
{{- if not (kindIs "invalid" $f.object_size_greater_than) }}{{ $_ := set $out "objectSizeGreaterThan" (int64 $f.object_size_greater_than) }}{{ end -}}
{{- if not (kindIs "invalid" $f.object_size_less_than) }}{{ $_ := set $out "objectSizeLessThan" (int64 $f.object_size_less_than) }}{{ end -}}
{{- if eq (len $tags) 1 }}{{ $_ := set $out "tag" (index $tags 0) }}{{ end -}}
{{- else if gt $count 1 -}}
{{- $and := dict -}}
{{- if $f.prefix }}{{ $_ := set $and "prefix" $f.prefix }}{{ end -}}
{{- if $f.access_point }}{{ $_ := set $and "accessPointARN" $f.access_point }}{{ end -}}
{{- if not (kindIs "invalid" $f.object_size_greater_than) }}{{ $_ := set $and "objectSizeGreaterThan" (int64 $f.object_size_greater_than) }}{{ end -}}
{{- if not (kindIs "invalid" $f.object_size_less_than) }}{{ $_ := set $and "objectSizeLessThan" (int64 $f.object_size_less_than) }}{{ end -}}
{{- if $tags }}{{ $_ := set $and "tags" $tags }}{{ end -}}
{{- $_ := set $out "and" $and -}}
{{- end -}}
{{- toJson $out -}}
{{- end -}}

{{- /*
s3.policyStatements: builds the combined bucket policy Statement list from
(list $root $name $arn).
*/ -}}
{{- define "s3.policyStatements" -}}
{{- $root := index . 0 -}}
{{- $name := index . 1 -}}
{{- $arn := index . 2 -}}
{{- $v := $root.Values -}}
{{- $st := list -}}

{{- /* ELB regional service accounts (regions created before August 2022). */ -}}
{{- $elbAccounts := dict
  "us-east-1" "127311923021"
  "us-east-2" "033677994240"
  "us-west-1" "027434742980"
  "us-west-2" "797873946194"
  "af-south-1" "098369216593"
  "ap-east-1" "754344448648"
  "ap-south-1" "718504428378"
  "ap-northeast-1" "582318560864"
  "ap-northeast-2" "600734575887"
  "ap-northeast-3" "383597477331"
  "ap-southeast-1" "114774131450"
  "ap-southeast-2" "783225319266"
  "ap-southeast-3" "589379963580"
  "ca-central-1" "985666609251"
  "eu-central-1" "054676820928"
  "eu-west-1" "156460612806"
  "eu-west-2" "652711504416"
  "eu-west-3" "009996457667"
  "eu-south-1" "635631232127"
  "eu-north-1" "897822967062"
  "me-south-1" "076674570225"
  "sa-east-1" "507241528517"
  "us-gov-west-1" "048591011584"
  "us-gov-east-1" "190560391635"
  "cn-north-1" "638102146993"
  "cn-northwest-1" "037604701340"
-}}

{{- if $v.attach_elb_log_delivery_policy -}}
{{- if and $v.region (hasKey $elbAccounts $v.region) -}}
{{- $acct := index $elbAccounts $v.region -}}
{{- $st = append $st (dict "Sid" (printf "ELBRegion%s" (title $v.region)) "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" (dict "AWS" (printf "arn:%s:iam::%s:root" $v.partition $acct))) -}}
{{- end -}}
{{- $st = append $st (dict "Sid" "" "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" (dict "Service" "logdelivery.elasticloadbalancing.amazonaws.com")) -}}
{{- end -}}

{{- if $v.attach_lb_log_delivery_policy -}}
{{- $cWrite := dict "StringEquals" (dict "s3:x-amz-acl" "bucket-owner-full-control") -}}
{{- $cAcl := dict -}}
{{- with $v.lb_log_delivery_policy_source_organizations -}}
{{- $_ := set $cWrite.StringEquals "aws:ResourceOrgID" . -}}
{{- $_ := set $cAcl "StringEquals" (dict "aws:ResourceOrgID" .) -}}
{{- end -}}
{{- $st = append $st (dict "Sid" "AlbNlbLogDeliveryWrite" "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" (dict "Service" "delivery.logs.amazonaws.com") "Condition" $cWrite) -}}
{{- $aclStmt := dict "Sid" "AlbNlbLogDeliveryAclCheck" "Effect" "Allow" "Action" (list "s3:GetBucketAcl" "s3:ListBucket") "Resource" (list $arn) "Principal" (dict "Service" "delivery.logs.amazonaws.com") -}}
{{- if $cAcl }}{{ $_ := set $aclStmt "Condition" $cAcl }}{{ end -}}
{{- $st = append $st $aclStmt -}}
{{- end -}}

{{- if $v.attach_access_log_delivery_policy -}}
{{- $cWrite := dict -}}
{{- with $v.access_log_delivery_policy_source_buckets -}}
{{- $_ := set $cWrite "ForAnyValue:ArnLike" (dict "aws:SourceArn" .) -}}
{{- end -}}
{{- with $v.access_log_delivery_policy_source_accounts -}}
{{- $_ := set $cWrite "ForAnyValue:StringEquals" (dict "aws:SourceAccount" .) -}}
{{- end -}}
{{- $cAcl := dict -}}
{{- with $v.access_log_delivery_policy_source_organizations -}}
{{- $_ := set $cWrite "StringEquals" (dict "aws:ResourceOrgID" .) -}}
{{- $_ := set $cAcl "StringEquals" (dict "aws:ResourceOrgID" .) -}}
{{- end -}}
{{- $writeStmt := dict "Sid" "AWSAccessLogDeliveryWrite" "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" (dict "Service" "logging.s3.amazonaws.com") -}}
{{- if $cWrite }}{{ $_ := set $writeStmt "Condition" $cWrite }}{{ end -}}
{{- $st = append $st $writeStmt -}}
{{- $aclStmt := dict "Sid" "AWSAccessLogDeliveryAclCheck" "Effect" "Allow" "Action" (list "s3:GetBucketAcl") "Resource" (list $arn) "Principal" (dict "Service" "logging.s3.amazonaws.com") -}}
{{- if $cAcl }}{{ $_ := set $aclStmt "Condition" $cAcl }}{{ end -}}
{{- $st = append $st $aclStmt -}}
{{- end -}}

{{- if $v.attach_waf_log_delivery_policy -}}
{{- if not $v.account_id -}}
{{- fail "s3-bucket: attach_waf_log_delivery_policy needs account_id: a chart cannot look the account up (values.yaml, account_id)" -}}
{{- end -}}
{{- $acct := toString $v.account_id -}}
{{- $srcArn := printf "arn:%s:logs:*:%s:*" $v.partition $acct -}}
{{- $st = append $st (dict "Sid" "WafLogDeliveryWrite" "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/AWSLogs/%s/*" $arn $acct)) "Principal" (dict "Service" "delivery.logs.amazonaws.com") "Condition" (dict "StringEquals" (dict "s3:x-amz-acl" "bucket-owner-full-control" "aws:SourceAccount" $acct) "ArnLike" (dict "aws:SourceArn" $srcArn))) -}}
{{- $st = append $st (dict "Sid" "WafLogDeliveryAclCheck" "Effect" "Allow" "Action" (list "s3:GetBucketAcl") "Resource" (list $arn) "Principal" (dict "Service" "delivery.logs.amazonaws.com") "Condition" (dict "StringEquals" (dict "aws:SourceAccount" $acct) "ArnLike" (dict "aws:SourceArn" $srcArn))) -}}
{{- end -}}

{{- if $v.attach_cloudtrail_log_delivery_policy -}}
{{- $st = append $st (dict "Sid" "AWSCloudTrailAclCheck" "Effect" "Allow" "Action" (list "s3:GetBucketAcl") "Resource" (list $arn) "Principal" (dict "Service" "cloudtrail.amazonaws.com")) -}}
{{- $st = append $st (dict "Sid" "AWSCloudTrailWrite" "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/AWSLogs/*" $arn)) "Principal" (dict "Service" "cloudtrail.amazonaws.com") "Condition" (dict "StringEquals" (dict "s3:x-amz-acl" "bucket-owner-full-control"))) -}}
{{- end -}}

{{- if or $v.attach_inventory_destination_policy $v.attach_analytics_destination_policy -}}
{{- $srcArns := list -}}
{{- $srcAccts := list -}}
{{- if $v.inventory_self_source_destination -}}
{{- if not $v.account_id }}{{ fail "s3-bucket: inventory_self_source_destination needs account_id" }}{{ end -}}
{{- $srcArns = append $srcArns $arn -}}
{{- $srcAccts = append $srcAccts (toString $v.account_id) -}}
{{- else -}}
{{- with $v.inventory_source_bucket_arn }}{{ $srcArns = append $srcArns . }}{{ end -}}
{{- with $v.inventory_source_account_id }}{{ $srcAccts = append $srcAccts (toString .) }}{{ end -}}
{{- end -}}
{{- if $v.analytics_self_source_destination -}}
{{- if not $v.account_id }}{{ fail "s3-bucket: analytics_self_source_destination needs account_id" }}{{ end -}}
{{- $srcArns = append $srcArns $arn -}}
{{- $srcAccts = append $srcAccts (toString $v.account_id) -}}
{{- else -}}
{{- with $v.analytics_source_bucket_arn }}{{ $srcArns = append $srcArns . }}{{ end -}}
{{- with $v.analytics_source_account_id }}{{ $srcAccts = append $srcAccts (toString .) }}{{ end -}}
{{- end -}}
{{- $srcArns = uniq $srcArns -}}
{{- $srcAccts = uniq $srcAccts -}}
{{- $st = append $st (dict "Sid" "destinationInventoryAndAnalyticsPolicy" "Effect" "Allow" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" (dict "Service" "s3.amazonaws.com") "Condition" (dict "ArnLike" (dict "aws:SourceArn" $srcArns) "StringEquals" (dict "aws:SourceAccount" $srcAccts "s3:x-amz-acl" "bucket-owner-full-control"))) -}}
{{- end -}}

{{- range $p := list (list $v.attach_require_latest_tls_policy "denyOutdatedTLS" "NumericLessThan" "s3:TlsVersion" "1.2") (list $v.attach_deny_insecure_transport_policy "denyInsecureTransport" "Bool" "aws:SecureTransport" "false") -}}
{{- if index $p 0 -}}
{{- $st = append $st (dict "Sid" (index $p 1) "Effect" "Deny" "Action" (list "s3:*") "Resource" (list $arn (printf "%s/*" $arn)) "Principal" "*" "Condition" (dict (index $p 2) (dict (index $p 3) (list (index $p 4))))) -}}
{{- end -}}
{{- end -}}

{{- if $v.attach_deny_incorrect_encryption_headers -}}
{{- $sseAlg := (((($v.server_side_encryption_configuration | default dict).rule | default dict).apply_server_side_encryption_by_default | default dict).sse_algorithm | default "") -}}
{{- $expected := ternary "aws:kms" "AES256" (eq $sseAlg "aws:kms") -}}
{{- $st = append $st (dict "Sid" "denyIncorrectEncryptionHeaders" "Effect" "Deny" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" "*" "Condition" (dict "StringNotEquals" (dict "s3:x-amz-server-side-encryption" (list $expected)))) -}}
{{- end -}}

{{- if $v.attach_deny_incorrect_kms_key_sse -}}
{{- if not $v.allowed_kms_key_arn -}}
{{- fail "s3-bucket: allowed_kms_key_arn is required with attach_deny_incorrect_kms_key_sse: true" -}}
{{- end -}}
{{- $st = append $st (dict "Sid" "denyIncorrectKmsKeySse" "Effect" "Deny" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" "*" "Condition" (dict "StringNotEquals" (dict "s3:x-amz-server-side-encryption-aws-kms-key-id" (list $v.allowed_kms_key_arn)))) -}}
{{- end -}}

{{- if $v.attach_deny_unencrypted_object_uploads -}}
{{- $st = append $st (dict "Sid" "denyUnencryptedObjectUploads" "Effect" "Deny" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" "*" "Condition" (dict "Null" (dict "s3:x-amz-server-side-encryption" (list true)))) -}}
{{- end -}}

{{- if $v.attach_deny_ssec_encrypted_object_uploads -}}
{{- $st = append $st (dict "Sid" "denySSECEncryptedObjectUploads" "Effect" "Deny" "Action" (list "s3:PutObject") "Resource" (list (printf "%s/*" $arn)) "Principal" "*" "Condition" (dict "Null" (dict "s3:x-amz-server-side-encryption-customer-algorithm" (list false)))) -}}
{{- end -}}

{{- if and $v.attach_policy $v.policy -}}
{{- $doc := $v.policy | replace "_S3_BUCKET_ID_" $name | replace "_S3_BUCKET_ARN_" $arn -}}
{{- if contains "_AWS_ACCOUNT_ID_" $doc -}}
{{- if not $v.account_id -}}
{{- fail "s3-bucket: policy uses _AWS_ACCOUNT_ID_, and account_id is unset: a chart cannot look the account up (values.yaml, account_id)" -}}
{{- end -}}
{{- $doc = $doc | replace "_AWS_ACCOUNT_ID_" (toString $v.account_id) -}}
{{- end -}}
{{- $parsed := $doc | fromJson -}}
{{- if hasKey $parsed "Error" -}}
{{- fail (printf "s3-bucket: policy is not a JSON policy document: %s" $parsed.Error) -}}
{{- end -}}
{{- $ps := $parsed.Statement -}}
{{- if kindIs "map" $ps }}{{ $ps = list $ps }}{{ end -}}
{{- $st = concat $st ($ps | default list) -}}
{{- end -}}

{{- toJson $st -}}
{{- end -}}
