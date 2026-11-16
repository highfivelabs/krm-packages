{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
Helpers and objects shared by the Application Load Balancer charts
(net-lb-app-ext, net-lb-app-int): each carries an identical copy, as the
VPN charts carry _router.tpl. Ported from cloud-foundation-fabric
modules/net-lb-app-ext and modules/net-lb-app-int (v60.0.0): their
backend-service.tf, backends.tf, groups.tf, health-check.tf, negs.tf and
urlmap.tf, Copyright 2026 Google LLC, Apache-2.0; written after
net-lb-app-int-cross-region's helpers.

What differs between the charts comes from the chart's own
"netlbapp.cfg" (a JSON object):
  region         "" for a global load balancer, else its region
  scheme         the backend services' load balancing scheme
  portNameLower  true: a backend service without port_name takes
                 lower(protocol), as net-lb-app-ext does
  negProject     true: every NEG type takes its entry's project_id
                 (net-lb-app-int); false: only serverless ones do
  negEntryDesc   true: a NEG's description is its entry's, else the
                 chart's (net-lb-app-ext); false: the chart's
  pscDesc        true: PSC NEGs take a description
  hcRegional     true: every health check is regional
                 (net-lb-app-ext-regional); else one is regional when
                 its entry sets is_regional
  internetByEndpoint true: an internet NEG is INTERNET_FQDN_PORT when an
                 endpoint has an fqdn (or it has none), its endpoints
                 {fqdn, ip_address, port} (net-lb-app-ext-regional); else
                 use_fqdn decides, its endpoints {destination, port}
*/ -}}

{{- /* netlbapp.c: the chart's configuration as a dict. */ -}}
{{- define "netlbapp.c" -}}
{{- include "netlbapp.cfg" . -}}
{{- end -}}

{{- /* netlbapp.link: a resource link without the Compute API's URL prefix,
as Config Connector's external references take it. */ -}}
{{- define "netlbapp.link" -}}
{{- trimPrefix "https://www.googleapis.com/compute/beta/" (trimPrefix "https://www.googleapis.com/compute/v1/" (toString .)) -}}
{{- end -}}

{{- /* netlbapp.project: an entry's project_id, else the chart's. (list $ entry) */ -}}
{{- define "netlbapp.project" -}}
{{- $e := index . 1 | default dict -}}
{{- $e.project_id | default (index . 0).Values.project_id -}}
{{- end -}}

{{- /* netlbapp.location: the region of a regional load balancer, else
"global". */ -}}
{{- define "netlbapp.location" -}}
{{- $c := include "netlbapp.c" . | fromJson -}}
{{- $c.region | default "global" -}}
{{- end -}}

{{- /* netlbapp.scope: the path segment of IDs: global or regions/<region>. */ -}}
{{- define "netlbapp.scope" -}}
{{- $c := include "netlbapp.c" . | fromJson -}}
{{- if $c.region }}regions/{{ $c.region }}{{ else }}global{{ end -}}
{{- end -}}

{{- /* netlbapp.negName: the cloud name of the network endpoint group of a
key, <name>-<key> with the namePrefix. (list $ key) */ -}}
{{- define "netlbapp.negName" -}}
{{- include "kcc.cloudName" (list (index . 0) "network endpoint group" (printf "%s-%s" (index . 0).Values.name (index . 1)) 63 "letter") -}}
{{- end -}}

{{- /* netlbapp.bsName: the cloud name of the backend service of a key:
the entry's name, else <name>-<key>, with the namePrefix. (list $ key) */ -}}
{{- define "netlbapp.bsName" -}}
{{- $root := index . 0 -}}{{- $k := index . 1 -}}
{{- $b := index ($root.Values.backend_service_configs | default dict) $k -}}
{{- include "kcc.cloudName" (list $root "backend service" ($b.name | default (printf "%s-%s" $root.Values.name $k)) 63 "letter") -}}
{{- end -}}

{{- /* netlbapp.bucketName: the cloud name of the backend bucket of a key.
(list $ key) */ -}}
{{- define "netlbapp.bucketName" -}}
{{- $root := index . 0 -}}{{- $k := index . 1 -}}
{{- $b := index ($root.Values.backend_buckets_config | default dict) $k -}}
{{- include "kcc.cloudName" (list $root "backend bucket" ($b.name | default (printf "%s-%s" $root.Values.name $k)) 63 "letter") -}}
{{- end -}}

{{- /* netlbapp.bsRef: a reference to a backend service: one of the chart's
by key (by Kubernetes name), else an existing one's ID or self link.
(list $ service) */ -}}
{{- define "netlbapp.bsRef" -}}
{{- $root := index . 0 -}}{{- $s := index . 1 -}}
{{- if hasKey ($root.Values.backend_service_configs | default dict) $s -}}
name: {{ include "kcc.name" (list $root.Release.Name "bs" $s) }}
{{- else -}}
external: {{ include "netlbapp.link" $s | quote }}
{{- end -}}
{{- end -}}

{{- /* netlbapp.serviceRef: a URL map service, a backend service or a
backend bucket: one of the chart's by key, else an existing one's ID or
self link (a bucket when it holds /backendBuckets/). (list $ service) */ -}}
{{- define "netlbapp.serviceRef" -}}
{{- $root := index . 0 -}}{{- $s := index . 1 -}}
{{- if hasKey ($root.Values.backend_buckets_config | default dict) $s -}}
backendBucketRef:
  name: {{ include "kcc.name" (list $root.Release.Name "bucket" $s) }}
{{- else if and (not (hasKey ($root.Values.backend_service_configs | default dict) $s)) (contains "/backendBuckets/" $s) -}}
backendBucketRef:
  external: {{ include "netlbapp.link" $s | quote }}
{{- else -}}
backendServiceRef:
  {{- include "netlbapp.bsRef" (list $root $s) | nindent 2 }}
{{- end -}}
{{- end -}}

{{- /* netlbapp.bsID: a backend service's or bucket's ID, for fields that
take a URL rather than a reference:
projects/<project>/<scope>/backendServices/<name> (or
projects/<project>/global/backendBuckets/<name>) for one of the chart's by
key, else the value as given. (list $ service) */ -}}
{{- define "netlbapp.bsID" -}}
{{- $root := index . 0 -}}{{- $s := index . 1 -}}
{{- $bss := $root.Values.backend_service_configs | default dict -}}
{{- $bbs := $root.Values.backend_buckets_config | default dict -}}
{{- if hasKey $bss $s -}}
{{- printf "projects/%s/%s/backendServices/%s" (include "netlbapp.project" (list $root (index $bss $s))) (include "netlbapp.scope" $root) (include "netlbapp.bsName" (list $root $s)) -}}
{{- else if hasKey $bbs $s -}}
{{- printf "projects/%s/global/backendBuckets/%s" (include "netlbapp.project" (list $root (index $bbs $s))) (include "netlbapp.bucketName" (list $root $s)) -}}
{{- else -}}
{{- include "netlbapp.link" $s -}}
{{- end -}}
{{- end -}}

{{- /* netlbapp.duration: {seconds, nanos} as Config Connector's duration
(seconds is a string in ComputeURLMap). (list duration secondsAsString) */ -}}
{{- define "netlbapp.duration" -}}
{{- $d := index . 0 -}}
{{- if index . 1 }}
seconds: {{ $d.seconds | toString | quote }}
{{- else }}
seconds: {{ $d.seconds }}
{{- end }}
{{- if not (kindIs "invalid" $d.nanos) }}
nanos: {{ $d.nanos }}
{{- end }}
{{- end -}}

{{- /* netlbapp.headerAction: fabric's {request_add, request_remove,
response_add, response_remove} as headerAction. */ -}}
{{- define "netlbapp.headerAction" -}}
{{- $h := . -}}
{{- with $h.request_add }}
requestHeadersToAdd:
  {{- range $n := keys . | sortAlpha }}
  {{- $a := index $h.request_add $n }}
  - headerName: {{ $n | quote }}
    headerValue: {{ $a.value | quote }}
    replace: {{ ternary true $a.replace (kindIs "invalid" $a.replace) }}
  {{- end }}
{{- end }}
{{- with $h.request_remove }}
requestHeadersToRemove: {{ toJson . }}
{{- end }}
{{- with $h.response_add }}
responseHeadersToAdd:
  {{- range $n := keys . | sortAlpha }}
  {{- $a := index $h.response_add $n }}
  - headerName: {{ $n | quote }}
    headerValue: {{ $a.value | quote }}
    replace: {{ ternary true $a.replace (kindIs "invalid" $a.replace) }}
  {{- end }}
{{- end }}
{{- with $h.response_remove }}
responseHeadersToRemove: {{ toJson . }}
{{- end }}
{{- end -}}

{{- /* netlbapp.redirect: fabric's {host, https, path, prefix,
response_code, strip_query} as a URL redirect. */ -}}
{{- define "netlbapp.redirect" -}}
{{- $r := . -}}
{{- range $f := list (list "host" "hostRedirect") (list "https" "httpsRedirect") (list "path" "pathRedirect") (list "prefix" "prefixRedirect") (list "response_code" "redirectResponseCode") (list "strip_query" "stripQuery") }}
{{- $val := index $r (index $f 0) }}
{{- if not (kindIs "invalid" $val) }}
{{ index $f 1 }}: {{ toJson $val }}
{{- end }}
{{- end }}
{{- end -}}

{{- /* netlbapp.errorPolicy: fabric's {error_service, error_response_rules}
as a custom error response policy. The error service is the key of one of
the chart's backend buckets (or services), else an existing one's ID or
self link. (list $ policy) */ -}}
{{- define "netlbapp.errorPolicy" -}}
{{- $root := index . 0 -}}{{- $p := index . 1 -}}
{{- with $p.error_service }}
errorServiceRef:
  {{- if hasKey ($root.Values.backend_buckets_config | default dict) . }}
  name: {{ include "kcc.name" (list $root.Release.Name "bucket" .) }}
  {{- else }}
  external: {{ include "netlbapp.bsID" (list $root .) | quote }}
  {{- end }}
{{- end }}
{{- with $p.error_response_rules }}
{{- $rules := list }}
{{- range $r := . }}
{{- $o := dict }}
{{- if not (kindIs "invalid" $r.match_response_codes) }}{{ $_ := set $o "matchResponseCodes" $r.match_response_codes }}{{ end }}
{{- if not (kindIs "invalid" $r.override_response_code) }}{{ $_ := set $o "overrideResponseCode" (int $r.override_response_code) }}{{ end }}
{{- if not (kindIs "invalid" $r.path) }}{{ $_ := set $o "path" $r.path }}{{ end }}
{{- $rules = append $rules $o }}
{{- end }}
errorResponseRule:
  {{- toYaml $rules | nindent 2 }}
{{- end }}
{{- end -}}

{{- /* netlbapp.routeAction: fabric's route action as routeAction (or
defaultRouteAction). (list $ action) */ -}}
{{- define "netlbapp.routeAction" -}}
{{- $root := index . 0 -}}{{- $ra := index . 1 -}}
{{- with $ra.cors_policy }}
corsPolicy:
  {{- range $f := list (list "allow_credentials" "allowCredentials") (list "allow_headers" "allowHeaders") (list "allow_methods" "allowMethods") (list "allow_origin_regexes" "allowOriginRegexes") (list "allow_origins" "allowOrigins") (list "disabled" "disabled") (list "expose_headers" "exposeHeaders") }}
  {{- $val := index $ra.cors_policy (index $f 0) }}
  {{- if not (kindIs "invalid" $val) }}
  {{ index $f 1 }}: {{ toJson $val }}
  {{- end }}
  {{- end }}
  {{- if not (kindIs "invalid" .max_age) }}
  maxAge: {{ int .max_age }}
  {{- end }}
{{- end }}
{{- with $ra.fault_injection_policy }}
faultInjectionPolicy:
  {{- with .abort }}
  abort:
    httpStatus: {{ .status }}
    percentage: {{ .percentage }}
  {{- end }}
  {{- with .delay }}
  delay:
    fixedDelay:
      seconds: {{ .fixed.seconds | toString | quote }}
      nanos: {{ .fixed.nanos }}
    percentage: {{ .percentage }}
  {{- end }}
{{- end }}
{{- with $ra.request_mirror_backend }}
requestMirrorPolicy:
  backendServiceRef:
    {{- include "netlbapp.bsRef" (list $root .) | trim | nindent 4 }}
{{- end }}
{{- with $ra.retry_policy }}
retryPolicy:
  numRetries: {{ .num_retries }}
  {{- with .retry_conditions }}
  retryConditions: {{ toJson . }}
  {{- end }}
  {{- with .per_try_timeout }}
  perTryTimeout:
    {{- include "netlbapp.duration" (list . true) | trim | nindent 4 }}
  {{- end }}
{{- end }}
{{- with $ra.timeout }}
timeout:
  {{- include "netlbapp.duration" (list . true) | trim | nindent 2 }}
{{- end }}
{{- with $ra.url_rewrite }}
urlRewrite:
  {{- with .host }}
  hostRewrite: {{ . | quote }}
  {{- end }}
  {{- with .path_prefix }}
  pathPrefixRewrite: {{ . | quote }}
  {{- end }}
  {{- with .path_template }}
  pathTemplateRewrite: {{ . | quote }}
  {{- end }}
{{- end }}
{{- with $ra.weighted_backend_services }}
weightedBackendServices:
  {{- range $s := keys . | sortAlpha }}
  {{- $w := index $ra.weighted_backend_services $s }}
  - backendServiceRef:
      {{- include "netlbapp.bsRef" (list $root $s) | trim | nindent 6 }}
    weight: {{ $w.weight }}
    {{- with $w.header_action }}
    headerAction:
      {{- include "netlbapp.headerAction" . | trim | nindent 6 }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end -}}

{{- /* netlbapp.healthChecks: google_compute_health_check.default (and,
with is_regional, google_compute_region_health_check.default): one
ComputeHealthCheck per health_check_configs key (fabric's default: key
default, HTTP on the serving port), named <name>-<key> unless it has a
name, in the entry's project or the chart's; global, or in the load
balancer's region when the entry sets is_regional. */ -}}
{{- define "netlbapp.healthChecks" -}}
{{- $v := .Values }}
{{- $c := include "netlbapp.c" . | fromJson }}
{{- $hcs := .Values.health_check_configs }}
{{- if kindIs "invalid" $hcs }}{{ $hcs = dict "default" (dict "http" (dict "port_specification" "USE_SERVING_PORT")) }}{{ end }}
{{- range $k := keys $hcs | sortAlpha }}
{{- $hc := index $hcs $k }}
{{- $types := list }}
{{- range $t := list "grpc" "http" "http2" "https" "tcp" "ssl" }}{{ if not (kindIs "invalid" (index $hc $t)) }}{{ $types = append $types $t }}{{ end }}{{ end }}
{{- if gt (len $types) 1 }}
{{- fail (printf "%s: Only one health check type can be configured at a time." $.Chart.Name) }}
{{- end }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeHealthCheck
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "hc" $k)) "id" (printf "health_check:%s" $k) "project" (include "netlbapp.project" (list $ $hc)) "abandon" (include "kcc.abandon" $)) | nindent 2 }}
spec:
  resourceID: {{ include "kcc.cloudName" (list $ "health check" ($hc.name | default (printf "%s-%s" $v.name $k)) 63 "letter") | quote }}
  location: {{ ternary $c.region "global" (and (or $c.hcRegional (eq (toString $hc.is_regional) "true")) (not (empty $c.region))) | quote }}
  description: {{ ternary "Terraform managed." $hc.description (kindIs "invalid" $hc.description) | quote }}
  {{- range $f := list (list "check_interval_sec" "checkIntervalSec") (list "healthy_threshold" "healthyThreshold") (list "timeout_sec" "timeoutSec") (list "unhealthy_threshold" "unhealthyThreshold") }}
  {{- if not (kindIs "invalid" (index $hc (index $f 0))) }}
  {{ index $f 1 }}: {{ index $hc (index $f 0) }}
  {{- end }}
  {{- end }}
  {{- range $t := $types }}
  {{- $hcc := index $hc $t | default dict }}
  {{ get (dict "grpc" "grpcHealthCheck" "http" "httpHealthCheck" "http2" "http2HealthCheck" "https" "httpsHealthCheck" "tcp" "tcpHealthCheck" "ssl" "sslHealthCheck") $t }}:
    {{- $fields := dict "port" "port" "port_name" "portName" "port_specification" "portSpecification" "service_name" "grpcServiceName" "host" "host" "proxy_header" "proxyHeader" "request_path" "requestPath" "request" "request" "response" "response" }}
    {{- $out := dict }}
    {{- range $from, $to := $fields }}{{ if not (kindIs "invalid" (index $hcc $from)) }}{{ $_ := set $out $to (index $hcc $from) }}{{ end }}{{ end }}
    {{- if $out }}
    {{- toYaml $out | nindent 4 }}
    {{- else }} {}
    {{- end }}
  {{- end }}
  {{- if eq (toString $hc.enable_logging) "true" }}
  logConfig:
    enable: true
  {{- end }}
{{- end }}
{{- end -}}

{{- /* netlbapp.groups: google_compute_instance_group.default: one unmanaged
ComputeInstanceGroup per group_configs entry, named <name>-<key> unless
it has a name, in the entry's project or the chart's. */ -}}
{{- define "netlbapp.groups" -}}
{{- $v := .Values }}
{{- $abandon := include "kcc.abandon" . }}
{{- range $k := keys ($v.group_configs | default dict) | sortAlpha }}
{{- $g := index $v.group_configs $k }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeInstanceGroup
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "group" $k)) "id" (printf "group:%s" $k) "project" (include "netlbapp.project" (list $ $g)) "abandon" $abandon) | nindent 2 }}
spec:
  resourceID: {{ include "kcc.cloudName" (list $ "instance group" ($g.name | default (printf "%s-%s" $v.name $k)) 63 "letter") | quote }}
  zone: {{ $g.zone | quote }}
  description: {{ $g.description | default "Terraform managed." | quote }}
  {{- with $g.instances }}
  instances:
    {{- range . }}
    - external: {{ include "netlbapp.link" . | quote }}
    {{- end }}
  {{- end }}
  {{- with $g.named_ports }}
  namedPort:
    {{- range $n := keys . | sortAlpha }}
    - name: {{ $n | quote }}
      port: {{ index $g.named_ports $n }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end -}}

{{- /* netlbapp.cdnPolicy: fabric's cdn_policy as cdnPolicy (backend
services and buckets). */ -}}
{{- define "netlbapp.cdnPolicy" -}}
{{- $p := . -}}
{{- range $f := list (list "cache_mode" "cacheMode") (list "client_ttl" "clientTtl") (list "default_ttl" "defaultTtl") (list "max_ttl" "maxTtl") (list "negative_caching" "negativeCaching") (list "request_coalescing" "requestCoalescing") (list "serve_while_stale" "serveWhileStale") (list "signed_url_cache_max_age_sec" "signedUrlCacheMaxAgeSec") }}
{{- $val := index $p (index $f 0) }}
{{- if not (kindIs "invalid" $val) }}
{{ index $f 1 }}: {{ toJson $val }}
{{- end }}
{{- end }}
{{- with $p.bypass_cache_on_request_headers }}
bypassCacheOnRequestHeaders:
  {{- range . }}
  - headerName: {{ . | quote }}
  {{- end }}
{{- end }}
{{- with $p.cache_key_policy }}
{{- $ck := . }}
cacheKeyPolicy:
  {{- range $f := list (list "include_host" "includeHost") (list "include_http_headers" "includeHttpHeaders") (list "include_named_cookies" "includeNamedCookies") (list "include_protocol" "includeProtocol") (list "include_query_string" "includeQueryString") (list "query_string_blacklist" "queryStringBlacklist") (list "query_string_whitelist" "queryStringWhitelist") }}
  {{- $val := index $ck (index $f 0) }}
  {{- if not (kindIs "invalid" $val) }}
  {{ index $f 1 }}: {{ toJson $val }}
  {{- end }}
  {{- end }}
{{- end }}
{{- with $p.negative_caching_policy }}
{{- $o := dict }}
{{- if not (kindIs "invalid" .code) }}{{ $_ := set $o "code" .code }}{{ end }}
{{- if not (kindIs "invalid" .ttl) }}{{ $_ := set $o "ttl" .ttl }}{{ end }}
negativeCachingPolicy:
  {{- toYaml (list $o) | nindent 2 }}
{{- end }}
{{- end -}}

{{- /* netlbapp.backendServices: google_compute_backend_service.default
(google_compute_region_backend_service.default for a regional load
balancer): one ComputeBackendService per backend_service_configs key,
named <name>-<key> unless it has a name. A backend's group is the key of
one of the chart's instance groups or network endpoint groups, or an
existing group's ID or self link; a health check is the key of one of the
chart's, or an existing one's ID or self link. */ -}}
{{- define "netlbapp.backendServices" -}}
{{- $v := .Values }}
{{- $c := include "netlbapp.c" . | fromJson }}
{{- $loc := include "netlbapp.location" . }}
{{- $groups := $v.group_configs | default dict }}
{{- $negs := $v.neg_configs | default dict }}
{{- $hcs := .Values.health_check_configs }}
{{- if kindIs "invalid" $hcs }}{{ $hcs = dict "default" (dict "http" (dict "port_specification" "USE_SERVING_PORT")) }}{{ end }}
{{- range $k := keys ($v.backend_service_configs | default dict) | sortAlpha }}
{{- $bs := index $v.backend_service_configs $k }}
{{- with $bs.log_config }}
{{- include "kcc.unsupported" (list $ (printf "backend_service_configs.%s.log_config.optional_mode" $k) .optional_mode "ComputeBackendService's logConfig has no optionalMode field") }}
{{- include "kcc.unsupported" (list $ (printf "backend_service_configs.%s.log_config.optional_fields" $k) .optional_fields "ComputeBackendService's logConfig has no optionalFields field") }}
{{- end }}
{{- if not (kindIs "invalid" $bs.tls_settings) }}
{{- include "kcc.unsupported" (list $ (printf "backend_service_configs.%s.tls_settings" $k) (dict "set" true) "ComputeBackendService has no tlsSettings field") }}
{{- end }}
{{- with $bs.security_settings }}
{{- include "kcc.unsupported" (list $ (printf "backend_service_configs.%s.security_settings.aws_v4_authentication" $k) .aws_v4_authentication "ComputeBackendService's securitySettings has no awsV4Authentication field") }}
{{- end }}
{{- $protocol := $bs.protocol | default $v.protocol | default "HTTP" }}
{{- $healthChecks := list "default" }}
{{- if not (kindIs "invalid" $bs.health_checks) }}{{ $healthChecks = $bs.health_checks }}{{ end }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeBackendService
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "bs" $k)) "id" (printf "backend_service:%s" $k) "project" (include "netlbapp.project" (list $ $bs)) "abandon" (include "kcc.abandon" $)) | nindent 2 }}
spec:
  resourceID: {{ include "netlbapp.bsName" (list $ $k) | quote }}
  location: {{ $loc | quote }}
  description: {{ ternary "Terraform managed." $bs.description (kindIs "invalid" $bs.description) | quote }}
  loadBalancingScheme: {{ $c.scheme | quote }}
  protocol: {{ $protocol | quote }}
  {{- with $healthChecks }}
  healthChecks:
    {{- range $h := . }}
    - healthCheckRef:
        {{- if hasKey $hcs $h }}
        name: {{ include "kcc.name" (list $.Release.Name "hc" $h) }}
        {{- else }}
        external: {{ include "netlbapp.link" $h | quote }}
        {{- end }}
    {{- end }}
  {{- end }}
  {{- if not (kindIs "invalid" $bs.affinity_cookie_ttl_sec) }}
  affinityCookieTtlSec: {{ $bs.affinity_cookie_ttl_sec }}
  {{- end }}
  {{- with $bs.compression_mode }}
  compressionMode: {{ . | quote }}
  {{- end }}
  {{- if not (kindIs "invalid" $bs.connection_draining_timeout_sec) }}
  connectionDrainingTimeoutSec: {{ $bs.connection_draining_timeout_sec }}
  {{- end }}
  {{- with $bs.custom_request_headers }}
  customRequestHeaders: {{ toJson . }}
  {{- end }}
  {{- with $bs.custom_response_headers }}
  customResponseHeaders: {{ toJson . }}
  {{- end }}
  {{- if not (kindIs "invalid" $bs.enable_cdn) }}
  enableCdn: {{ $bs.enable_cdn }}
  {{- end }}
  {{- if and $bs.locality_lb_policy (kindIs "invalid" $bs.locality_lb_policies) }}
  localityLbPolicy: {{ $bs.locality_lb_policy | quote }}
  {{- end }}
  {{- if $bs.port_name }}
  portName: {{ $bs.port_name | quote }}
  {{- else if $c.portNameLower }}
  portName: {{ lower $protocol | quote }}
  {{- end }}
  {{- with $bs.security_policy }}
  securityPolicyRef:
    external: {{ include "netlbapp.link" . | quote }}
  {{- end }}
  {{- with $bs.session_affinity }}
  sessionAffinity: {{ . | quote }}
  {{- end }}
  {{- if eq (toString $bs.enable_subsetting) "true" }}
  subsetting:
    policy: CONSISTENT_HASH_SUBSETTING
  {{- end }}
  {{- if not (kindIs "invalid" $bs.timeout_sec) }}
  timeoutSec: {{ $bs.timeout_sec }}
  {{- end }}
  {{- if $bs.backends }}
  backend:
    {{- $seen := dict }}
    {{- range $b := $bs.backends }}
    {{- if not (hasKey $seen $b.group) }}
    {{- $_ := set $seen $b.group true }}
    {{- if eq (toString $b.preferred) "true" }}
    {{- include "kcc.unsupported" (list $ (printf "backend_service_configs.%s.backends[].preferred" $k) (dict "set" true) "ComputeBackendService's backend has no preference field") }}
    {{- end }}
    {{- $mr := $b.max_rate | default dict }}
    {{- $mc := $b.max_connections | default dict }}
    - balancingMode: {{ $b.balancing_mode | default "UTILIZATION" | quote }}
      capacityScaler: {{ ternary 1 $b.capacity_scaler (kindIs "invalid" $b.capacity_scaler) }}
      description: {{ ternary "Terraform managed." $b.description (kindIs "invalid" $b.description) | quote }}
      group:
        {{- if hasKey $groups $b.group }}
        instanceGroupRef:
          name: {{ include "kcc.name" (list $.Release.Name "group" $b.group) }}
        {{- else if hasKey $negs $b.group }}
        {{- $n := index $negs $b.group }}
        networkEndpointGroupRef:
          {{- if not (kindIs "invalid" $n.internet) }}
          {{- if $c.region }}
          external: {{ printf "projects/%s/regions/%s/networkEndpointGroups/%s" (include "netlbapp.negProject" (list $ $n "internet")) ($n.internet.region | default $c.region) (include "netlbapp.negName" (list $ $b.group)) | quote }}
          {{- else }}
          external: {{ printf "projects/%s/global/networkEndpointGroups/%s" $v.project_id (include "netlbapp.negName" (list $ $b.group)) | quote }}
          {{- end }}
          {{- else if not (kindIs "invalid" $n.psc) }}
          external: {{ printf "projects/%s/regions/%s/networkEndpointGroups/%s" (include "netlbapp.negProject" (list $ $n "psc")) $n.psc.region (include "netlbapp.negName" (list $ $b.group)) | quote }}
          {{- else if or (not (kindIs "invalid" $n.cloudrun)) (not (kindIs "invalid" $n.cloudfunction)) (not (kindIs "invalid" $n.serverless_deployment)) }}
          {{- $sl := $n.cloudrun | default $n.cloudfunction | default $n.serverless_deployment | default dict }}
          external: {{ printf "projects/%s/regions/%s/networkEndpointGroups/%s" (include "netlbapp.project" (list $ $n)) $sl.region (include "netlbapp.negName" (list $ $b.group)) | quote }}
          {{- else }}
          name: {{ include "kcc.name" (list $.Release.Name "neg" $b.group) }}
          {{- end }}
        {{- else if contains "/instanceGroups/" $b.group }}
        instanceGroupRef:
          external: {{ include "netlbapp.link" $b.group | quote }}
        {{- else }}
        networkEndpointGroupRef:
          external: {{ include "netlbapp.link" $b.group | quote }}
        {{- end }}
      {{- if not (kindIs "invalid" $mc.per_group) }}
      maxConnections: {{ int $mc.per_group }}
      {{- end }}
      {{- if not (kindIs "invalid" $mc.per_endpoint) }}
      maxConnectionsPerEndpoint: {{ int $mc.per_endpoint }}
      {{- end }}
      {{- if not (kindIs "invalid" $mc.per_instance) }}
      maxConnectionsPerInstance: {{ int $mc.per_instance }}
      {{- end }}
      {{- if not (kindIs "invalid" $mr.per_group) }}
      maxRate: {{ int $mr.per_group }}
      {{- end }}
      {{- if not (kindIs "invalid" $mr.per_endpoint) }}
      maxRatePerEndpoint: {{ $mr.per_endpoint }}
      {{- end }}
      {{- if not (kindIs "invalid" $mr.per_instance) }}
      maxRatePerInstance: {{ $mr.per_instance }}
      {{- end }}
      {{- if not (kindIs "invalid" $b.max_utilization) }}
      maxUtilization: {{ $b.max_utilization }}
      {{- end }}
    {{- end }}
    {{- end }}
  {{- end }}
  {{- with $bs.cdn_policy }}
  cdnPolicy:
    {{- include "netlbapp.cdnPolicy" . | trim | nindent 4 }}
  {{- end }}
  {{- with $bs.circuit_breakers }}
  circuitBreakers:
    {{- range $f := list (list "max_connections" "maxConnections") (list "max_pending_requests" "maxPendingRequests") (list "max_requests" "maxRequests") (list "max_requests_per_connection" "maxRequestsPerConnection") (list "max_retries" "maxRetries") }}
    {{- if not (kindIs "invalid" (index $bs.circuit_breakers (index $f 0))) }}
    {{ index $f 1 }}: {{ index $bs.circuit_breakers (index $f 0) }}
    {{- end }}
    {{- end }}
    {{- with .connect_timeout }}
    connectTimeout:
      {{- include "netlbapp.duration" (list . false) | trim | nindent 6 }}
    {{- end }}
  {{- end }}
  {{- with $bs.consistent_hash }}
  consistentHash:
    {{- with .http_header_name }}
    httpHeaderName: {{ . | quote }}
    {{- end }}
    {{- if not (kindIs "invalid" .minimum_ring_size) }}
    minimumRingSize: {{ .minimum_ring_size }}
    {{- end }}
    {{- with .http_cookie }}
    httpCookie:
      {{- with .name }}
      name: {{ . | quote }}
      {{- end }}
      {{- with .path }}
      path: {{ . | quote }}
      {{- end }}
      {{- with .ttl }}
      ttl:
        {{- include "netlbapp.duration" (list . false) | trim | nindent 8 }}
      {{- end }}
    {{- end }}
  {{- end }}
  {{- with $bs.iap_config }}
  iap:
    {{- with .oauth2_client_id }}
    oauth2ClientId: {{ . | quote }}
    {{- end }}
    {{- if not (kindIs "invalid" .oauth2_client_secret) }}
    oauth2ClientSecret: {{ include "kcc.secretRef" (list $ (printf "backend_service_configs.%s.iap_config.oauth2_client_secret" $k) .oauth2_client_secret) }}
    {{- end }}
    {{- with .oauth2_client_secret_sha256 }}
    oauth2ClientSecretSha256: {{ . | quote }}
    {{- end }}
  {{- end }}
  {{- with $bs.locality_lb_policies }}
  localityLbPolicies:
    {{- range $p := . }}
    {{- if and $p.policy $p.custom_policy }}
    {{- fail (printf "%s: When specified, all locality lb policies must have EITHER policy or custom_policy filled, not both." $.Chart.Name) }}
    {{- end }}
    {{- with $p.policy }}
    - policy:
        name: {{ .name | quote }}
    {{- end }}
    {{- with $p.custom_policy }}
    - customPolicy:
        name: {{ .name | quote }}
        {{- with .data }}
        data: {{ . | quote }}
        {{- end }}
    {{- end }}
    {{- end }}
  {{- end }}
  {{- with $bs.log_config }}
  logConfig:
    {{- if not (kindIs "invalid" .enable) }}
    enable: {{ .enable }}
    {{- end }}
    {{- if not (kindIs "invalid" .sample_rate) }}
    sampleRate: {{ .sample_rate }}
    {{- end }}
  {{- end }}
  {{- with $bs.outlier_detection }}
  outlierDetection:
    {{- range $f := list (list "consecutive_errors" "consecutiveErrors") (list "consecutive_gateway_failure" "consecutiveGatewayFailure") (list "enforcing_consecutive_errors" "enforcingConsecutiveErrors") (list "enforcing_consecutive_gateway_failure" "enforcingConsecutiveGatewayFailure") (list "enforcing_success_rate" "enforcingSuccessRate") (list "max_ejection_percent" "maxEjectionPercent") (list "success_rate_minimum_hosts" "successRateMinimumHosts") (list "success_rate_request_volume" "successRateRequestVolume") (list "success_rate_stdev_factor" "successRateStdevFactor") }}
    {{- if not (kindIs "invalid" (index $bs.outlier_detection (index $f 0))) }}
    {{ index $f 1 }}: {{ index $bs.outlier_detection (index $f 0) }}
    {{- end }}
    {{- end }}
    {{- with .base_ejection_time }}
    baseEjectionTime:
      {{- include "netlbapp.duration" (list . false) | trim | nindent 6 }}
    {{- end }}
    {{- with .interval }}
    interval:
      {{- include "netlbapp.duration" (list . false) | trim | nindent 6 }}
    {{- end }}
  {{- end }}
  {{- with $bs.security_settings }}
  {{- if or .client_tls_policy .subject_alt_names }}
  securitySettings:
    {{- with .client_tls_policy }}
    clientTLSPolicyRef:
      external: {{ . | quote }}
    {{- end }}
    {{- with .subject_alt_names }}
    subjectAltNames: {{ toJson . }}
    {{- end }}
  {{- end }}
  {{- end }}
{{- end }}
{{- end -}}

{{- /* netlbapp.negProject: the project of a NEG of a type: its entry's
project_id when the chart says so (negProject), else the chart's. (list $
entry type) */ -}}
{{- define "netlbapp.negProject" -}}
{{- $root := index . 0 -}}{{- $n := index . 1 -}}
{{- $c := include "netlbapp.c" $root | fromJson -}}
{{- if $c.negProject -}}
{{- include "netlbapp.project" (list $root $n) -}}
{{- else -}}
{{- $root.Values.project_id -}}
{{- end -}}
{{- end -}}

{{- /* netlbapp.negs: the network endpoint groups of neg_configs, one type
per entry, named <name>-<key>:
  gce, hybrid: google_compute_network_endpoint_group.default, a zonal
    ComputeNetworkEndpointGroup (GCE_VM_IP_PORT, NON_GCP_PRIVATE_IP_PORT),
    and google_compute_network_endpoint.default, one ComputeNetworkEndpoint
    (config-connector-extensions, compute.gcp.hfl.dev/v1beta1) per
    endpoint, keyed <neg>-<endpoint>;
  cloudrun, cloudfunction: a serverless ComputeRegionNetworkEndpointGroup;
  psc: a ComputeRegionNetworkEndpointGroup to a PSC target service;
  internet, global load balancer: google_compute_global_network_endpoint_group,
    config-connector-extensions' ComputeGlobalNetworkEndpointGroup
    (v1beta1), and google_compute_global_network_endpoint, Config
    Connector's ComputeGlobalNetworkEndpoint (v1alpha1, allow_alpha);
  internet, regional load balancer: a ComputeRegionNetworkEndpointGroup
    (INTERNET_FQDN_PORT or INTERNET_IP_PORT), and
    google_compute_region_network_endpoint, config-connector-extensions'
    ComputeRegionNetworkEndpoint (v1beta1). */ -}}
{{- define "netlbapp.negs" -}}
{{- $v := .Values }}
{{- $c := include "netlbapp.c" . | fromJson }}
{{- $abandon := include "kcc.abandon" . }}
{{- $vpc := $v.vpc_config | default dict }}
{{- range $k := keys ($v.neg_configs | default dict) | sortAlpha }}
{{- $n := index $v.neg_configs $k }}
{{- $types := list }}
{{- range $t := list "cloudfunction" "cloudrun" "serverless_deployment" "gce" "hybrid" "internet" "psc" }}{{ if not (kindIs "invalid" (index $n $t)) }}{{ $types = append $types $t }}{{ end }}{{ end }}
{{- if ne (len $types) 1 }}
{{- fail (printf "%s: Only one type of NEG can be configured at a time." $.Chart.Name) }}
{{- end }}
{{- $type := index $types 0 }}
{{- $negObject := include "kcc.name" (list $.Release.Name "neg" $k) }}
{{- $desc := $v.description }}
{{- if $c.negEntryDesc }}{{ $desc = $n.description | default $v.description }}{{ end }}
{{- if or (eq $type "gce") (eq $type "hybrid") }}
{{- $z := ternary $n.gce $n.hybrid (eq $type "gce") | default dict }}
{{- $negProject := include "netlbapp.negProject" (list $ $n $type) }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeNetworkEndpointGroup
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" $negObject "id" (printf "neg:%s" $k) "project" $negProject "abandon" $abandon) | nindent 2 }}
spec:
  resourceID: {{ include "netlbapp.negName" (list $ $k) | quote }}
  location: {{ $z.zone | quote }}
  {{- if not (kindIs "invalid" $desc) }}
  description: {{ $desc | quote }}
  {{- end }}
  networkEndpointType: {{ ternary "GCE_VM_IP_PORT" "NON_GCP_PRIVATE_IP_PORT" (eq $type "gce") }}
  networkRef:
    external: {{ include "netlbapp.link" ($z.network | default $vpc.network) | quote }}
  {{- if eq $type "gce" }}
  subnetworkRef:
    external: {{ include "netlbapp.link" ($z.subnetwork | default $vpc.subnetwork) | quote }}
  {{- end }}
{{- range $ek := keys ($z.endpoints | default dict) | sortAlpha }}
{{- $e := index $z.endpoints $ek }}
---
apiVersion: compute.gcp.hfl.dev/v1beta1
kind: ComputeNetworkEndpoint
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "ep" $k $ek)) "id" (printf "endpoint:%s-%s" $k $ek) "project" $negProject "abandon" $abandon) | nindent 2 }}
spec:
  projectRef:
    external: {{ $negProject | quote }}
  networkEndpointGroupRef:
    name: {{ $negObject }}
  zone: {{ $z.zone | quote }}
  ipAddress: {{ $e.ip_address | quote }}
  {{- with $e.instance }}
  instanceRef:
    external: {{ . | quote }}
  {{- end }}
  resourceID: {{ $e.port | toString | quote }}
{{- end }}
{{- else if eq $type "serverless_deployment" }}
{{- include "kcc.unsupported" (list $ (printf "neg_configs.%s.serverless_deployment" $k) (dict "set" true) "ComputeRegionNetworkEndpointGroup has no serverlessDeployment field") }}
{{- else if or (eq $type "cloudrun") (eq $type "cloudfunction") }}
{{- $sl := ternary $n.cloudrun $n.cloudfunction (eq $type "cloudrun") | default dict }}
{{- if and (eq $type "cloudrun") (kindIs "invalid" $sl.target_urlmask) (kindIs "invalid" $sl.target_service) }}
{{- fail (printf "%s: Cloud Run NEGs need either target service or target urlmask defined." $.Chart.Name) }}
{{- end }}
{{- if and (eq $type "cloudfunction") (kindIs "invalid" $sl.target_urlmask) (kindIs "invalid" $sl.target_function) }}
{{- fail (printf "%s: Cloud Function NEGs need either target function or target urlmask defined." $.Chart.Name) }}
{{- end }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeRegionNetworkEndpointGroup
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" $negObject "id" (printf "serverless_neg:%s" $k) "project" (include "netlbapp.project" (list $ $n)) "abandon" $abandon) | nindent 2 }}
spec:
  resourceID: {{ include "netlbapp.negName" (list $ $k) | quote }}
  region: {{ $sl.region | quote }}
  {{- if not (kindIs "invalid" $desc) }}
  description: {{ $desc | quote }}
  {{- end }}
  networkEndpointType: SERVERLESS
  {{- if eq $type "cloudrun" }}
  cloudRun:
    {{- with $sl.target_service }}
    serviceRef:
      external: {{ .name | quote }}
    {{- end }}
    {{- $tag := $sl.tag | default ($sl.target_service | default dict).tag }}
    {{- with $tag }}
    tag: {{ . | quote }}
    {{- end }}
    {{- with $sl.target_urlmask }}
    urlMask: {{ . | quote }}
    {{- end }}
  {{- else }}
  cloudFunction:
    {{- with $sl.target_function }}
    functionRef:
      external: {{ . | quote }}
    {{- end }}
    {{- with $sl.target_urlmask }}
    urlMask: {{ . | quote }}
    {{- end }}
  {{- end }}
{{- else if eq $type "internet" }}
{{- $i := $n.internet | default dict }}
{{- $negProject := include "netlbapp.negProject" (list $ $n "internet") }}
{{- $fqdn := ne (toString $i.use_fqdn) "false" }}
{{- if $c.internetByEndpoint }}
{{- $fqdn = empty $i.endpoints }}
{{- range $e := $i.endpoints | default dict }}
{{- if $e.fqdn }}{{ $fqdn = true }}{{ end }}
{{- if and (not $e.fqdn) (not $e.ip_address) }}
{{- fail (printf "%s: Internet NEG endpoints must specify either fqdn or ip_address." $.Chart.Name) }}
{{- end }}
{{- end }}
{{- end }}
{{- if empty $c.region }}
{{- include "kcc.alpha" (list $ (printf "neg_configs.%s.internet.endpoints" $k) $i.endpoints "ComputeGlobalNetworkEndpoint") }}
{{- end }}
{{- if $c.region }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeRegionNetworkEndpointGroup
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" $negObject "id" (printf "internet_neg:%s" $k) "project" $negProject "abandon" $abandon) | nindent 2 }}
spec:
  resourceID: {{ include "netlbapp.negName" (list $ $k) | quote }}
  region: {{ $i.region | quote }}
  description: {{ $n.description | default $v.description | quote }}
  networkEndpointType: {{ ternary "INTERNET_FQDN_PORT" "INTERNET_IP_PORT" $fqdn }}
  networkRef:
    external: {{ include "netlbapp.link" ($i.network | default $vpc.network) | quote }}
{{- range $ek := keys ($i.endpoints | default dict) | sortAlpha }}
{{- $e := index $i.endpoints $ek }}
---
apiVersion: compute.gcp.hfl.dev/v1beta1
kind: ComputeRegionNetworkEndpoint
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "ep" $k $ek)) "id" (printf "internet_endpoint:%s-%s" $k $ek) "project" $negProject "abandon" $abandon) | nindent 2 }}
spec:
  projectRef:
    external: {{ $negProject | quote }}
  region: {{ $i.region | quote }}
  regionNetworkEndpointGroupRef:
    name: {{ $negObject }}
  {{- if $fqdn }}
  fqdn: {{ $e.destination | default $e.fqdn | quote }}
  {{- else }}
  ipAddress: {{ $e.destination | default $e.ip_address | quote }}
  {{- end }}
  port: {{ $e.port }}
{{- end }}
{{- else }}
---
apiVersion: compute.gcp.hfl.dev/v1beta1
kind: ComputeGlobalNetworkEndpointGroup
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" $negObject "id" (printf "global_neg:%s" $k) "project" $negProject "abandon" $abandon) | nindent 2 }}
spec:
  projectRef:
    external: {{ $negProject | quote }}
  resourceID: {{ include "netlbapp.negName" (list $ $k) | quote }}
  description: {{ $n.description | default $v.description | quote }}
  networkEndpointType: {{ ternary "INTERNET_FQDN_PORT" "INTERNET_IP_PORT" $fqdn }}
{{- range $ek := keys ($i.endpoints | default dict) | sortAlpha }}
{{- $e := index $i.endpoints $ek }}
---
apiVersion: compute.cnrm.cloud.google.com/v1alpha1
kind: ComputeGlobalNetworkEndpoint
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "ep" $k $ek)) "id" (printf "global_endpoint:%s-%s" $k $ek) "project" $negProject "abandon" $abandon) | nindent 2 }}
spec:
  projectRef:
    external: {{ $negProject | quote }}
  globalNetworkEndpointGroup: {{ include "netlbapp.negName" (list $ $k) | quote }}
  {{- if $fqdn }}
  fqdn: {{ $e.destination | quote }}
  {{- else }}
  ipAddress: {{ $e.destination | quote }}
  {{- end }}
  resourceID: {{ $e.port | toString | quote }}
{{- end }}
{{- end }}
{{- else }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeRegionNetworkEndpointGroup
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" $negObject "id" (printf "psc_neg:%s" $k) "project" (include "netlbapp.negProject" (list $ $n "psc")) "abandon" $abandon) | nindent 2 }}
spec:
  resourceID: {{ include "netlbapp.negName" (list $ $k) | quote }}
  region: {{ $n.psc.region | quote }}
  {{- if and $c.pscDesc (not (kindIs "invalid" $desc)) }}
  description: {{ $desc | quote }}
  {{- end }}
  networkEndpointType: PRIVATE_SERVICE_CONNECT
  pscTargetService: {{ $n.psc.target_service | quote }}
  {{- with $n.psc.network }}
  networkRef:
    external: {{ include "netlbapp.link" . | quote }}
  {{- end }}
  {{- with $n.psc.subnetwork }}
  subnetworkRef:
    external: {{ include "netlbapp.link" . | quote }}
  {{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{- /* netlbapp.urlMap: google_compute_url_map.default (or
google_compute_region_url_map.default): one ComputeURLMap, named name. A
service anywhere in urlmap_config is the key of one of the chart's backend
services (or buckets), or an existing one's ID or self link (fabric's
lookup(local.backend_ids, ...)); a test's service is taken as given, as
fabric takes it. */ -}}
{{- define "netlbapp.urlMap" -}}
{{- $v := .Values }}
{{- $u := .Values.urlmap_config }}
{{- if kindIs "invalid" $u }}{{ $u = dict "default_service" "default" }}{{ end }}
{{- range $name, $m := $u.path_matchers | default dict }}
{{- range $r := $m.path_rules | default list }}
{{- include "kcc.unsupported" (list $ (printf "urlmap_config.path_matchers.%s.path_rules[].custom_error_response_policy" $name) $r.custom_error_response_policy "ComputeURLMap's pathRule has no customErrorResponsePolicy field") }}
{{- end }}
{{- range $r := $m.route_rules | default list }}
{{- include "kcc.unsupported" (list $ (printf "urlmap_config.path_matchers.%s.route_rules[].custom_error_response_policy" $name) $r.custom_error_response_policy "ComputeURLMap's routeRules has no customErrorResponsePolicy field") }}
{{- end }}
{{- end }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeURLMap
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name)) "id" "url_map" "project" $v.project_id "abandon" (include "kcc.abandon" .)) | nindent 2 }}
spec:
  resourceID: {{ include "kcc.resourceName" (list $ "urlMap" "URL map" $v.name 63 "letter") | quote }}
  location: {{ include "netlbapp.location" . | quote }}
  description: {{ ternary "Terraform managed." $u.description (kindIs "invalid" $u.description) | quote }}
  {{- with $u.default_service }}
  defaultService:
    {{- include "netlbapp.serviceRef" (list $ .) | trim | nindent 4 }}
  {{- end }}
  {{- with $u.default_custom_error_response_policy }}
  defaultCustomErrorResponsePolicy:
    {{- include "netlbapp.errorPolicy" (list $ .) | trim | nindent 4 }}
  {{- end }}
  {{- with $u.default_url_redirect }}
  defaultUrlRedirect:
    {{- include "netlbapp.redirect" . | trim | nindent 4 }}
  {{- end }}
  {{- with $u.default_route_action }}
  defaultRouteAction:
    {{- include "netlbapp.routeAction" (list $ .) | trim | nindent 4 }}
  {{- end }}
  {{- with $u.header_action }}
  headerAction:
    {{- include "netlbapp.headerAction" . | trim | nindent 4 }}
  {{- end }}
  {{- with $u.host_rules }}
  hostRule:
    {{- range . }}
    - hosts: {{ toJson .hosts }}
      pathMatcher: {{ .path_matcher | quote }}
      {{- with .description }}
      description: {{ . | quote }}
      {{- end }}
    {{- end }}
  {{- end }}
  {{- with $u.path_matchers }}
  pathMatcher:
    {{- range $name := keys . | sortAlpha }}
    {{- $m := index $u.path_matchers $name }}
    - name: {{ $name | quote }}
      {{- with $m.description }}
      description: {{ . | quote }}
      {{- end }}
      {{- with $m.default_service }}
      defaultService:
        {{- include "netlbapp.serviceRef" (list $ .) | trim | nindent 8 }}
      {{- end }}
      {{- with $m.default_custom_error_response_policy }}
      defaultCustomErrorResponsePolicy:
        {{- include "netlbapp.errorPolicy" (list $ .) | trim | nindent 8 }}
      {{- end }}
      {{- with $m.default_url_redirect }}
      defaultUrlRedirect:
        {{- include "netlbapp.redirect" . | trim | nindent 8 }}
      {{- end }}
      {{- with $m.default_route_action }}
      defaultRouteAction:
        {{- include "netlbapp.routeAction" (list $ .) | trim | nindent 8 }}
      {{- end }}
      {{- with $m.header_action }}
      headerAction:
        {{- include "netlbapp.headerAction" . | trim | nindent 8 }}
      {{- end }}
      {{- with $m.path_rules }}
      pathRule:
        {{- range $r := . }}
        - paths: {{ toJson $r.paths }}
          {{- with $r.service }}
          service:
            {{- include "netlbapp.serviceRef" (list $ .) | trim | nindent 12 }}
          {{- end }}
          {{- with $r.route_action }}
          routeAction:
            {{- include "netlbapp.routeAction" (list $ .) | trim | nindent 12 }}
          {{- end }}
          {{- with $r.url_redirect }}
          urlRedirect:
            {{- include "netlbapp.redirect" . | trim | nindent 12 }}
          {{- end }}
        {{- end }}
      {{- end }}
      {{- with $m.route_rules }}
      routeRules:
        {{- range $r := . }}
        - priority: {{ int $r.priority }}
          {{- with $r.service }}
          service: {{ include "netlbapp.bsID" (list $ .) | quote }}
          {{- end }}
          {{- with $r.header_action }}
          headerAction:
            {{- include "netlbapp.headerAction" . | trim | nindent 12 }}
          {{- end }}
          {{- with $r.match_rules }}
          matchRules:
            {{- range $mr := . }}
            {{- $pt := "" }}{{ with $mr.path }}{{ $pt = .type | default "prefix" }}{{ end }}
            - ignoreCase: {{ eq (toString $mr.ignore_case) "true" }}
              {{- if eq $pt "full" }}
              fullPathMatch: {{ $mr.path.value | quote }}
              {{- else if eq $pt "prefix" }}
              prefixMatch: {{ $mr.path.value | quote }}
              {{- else if eq $pt "regex" }}
              regexMatch: {{ $mr.path.value | quote }}
              {{- else if eq $pt "template" }}
              pathTemplateMatch: {{ $mr.path.value | quote }}
              {{- end }}
              {{- with $mr.headers }}
              headerMatches:
                {{- range $h := . }}
                {{- $ht := $h.type | default "present" }}
                - headerName: {{ $h.name | quote }}
                  invertMatch: {{ eq (toString $h.invert_match) "true" }}
                  {{- if not (kindIs "invalid" $h.value) }}
                  {{- if eq $ht "exact" }}
                  exactMatch: {{ $h.value | quote }}
                  {{- else if eq $ht "prefix" }}
                  prefixMatch: {{ $h.value | quote }}
                  {{- else if eq $ht "suffix" }}
                  suffixMatch: {{ $h.value | quote }}
                  {{- else if eq $ht "regex" }}
                  regexMatch: {{ $h.value | quote }}
                  {{- else if eq $ht "present" }}
                  presentMatch: {{ eq (toString $h.value) "true" }}
                  {{- end }}
                  {{- end }}
                  {{- if and (eq $ht "range") $h.range_value }}
                  rangeMatch:
                    rangeEnd: {{ int $h.range_value.end }}
                    rangeStart: {{ int $h.range_value.start }}
                  {{- end }}
                {{- end }}
              {{- end }}
              {{- with $mr.metadata_filters }}
              metadataFilters:
                {{- range $f := . }}
                - filterMatchCriteria: {{ ternary "MATCH_ALL" "MATCH_ANY" $f.match_all }}
                  filterLabels:
                    {{- range $ln := keys $f.labels | sortAlpha }}
                    - name: {{ $ln | quote }}
                      value: {{ index $f.labels $ln | quote }}
                    {{- end }}
                {{- end }}
              {{- end }}
              {{- with $mr.query_params }}
              queryParameterMatches:
                {{- range $q := . }}
                {{- $qt := $q.type | default "present" }}
                - name: {{ $q.name | quote }}
                  {{- if eq $qt "exact" }}
                  exactMatch: {{ $q.value | quote }}
                  {{- else if eq $qt "regex" }}
                  regexMatch: {{ $q.value | quote }}
                  {{- else }}
                  presentMatch: {{ eq (toString $q.value) "true" }}
                  {{- end }}
                {{- end }}
              {{- end }}
            {{- end }}
          {{- end }}
          {{- with $r.route_action }}
          routeAction:
            {{- include "netlbapp.routeAction" (list $ .) | trim | nindent 12 }}
          {{- end }}
          {{- with $r.url_redirect }}
          urlRedirect:
            {{- include "netlbapp.redirect" . | trim | nindent 12 }}
          {{- end }}
        {{- end }}
      {{- end }}
    {{- end }}
  {{- end }}
  {{- with $u.test }}
  test:
    {{- range $t := . }}
    - host: {{ $t.host | quote }}
      path: {{ $t.path | quote }}
      service:
        {{- include "netlbapp.serviceRef" (list $ $t.service) | trim | nindent 8 }}
      {{- with $t.description }}
      description: {{ . | quote }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- /* netlbapp.sslCertificates: google_compute_ssl_certificate.default (or
google_compute_region_ssl_certificate.default): one ComputeSSLCertificate
per ssl_certificates.create_configs key, named <name>-<key> (or the
entry's name). The certificate (public) is sent as given; the private key
is a reference to a Kubernetes Secret, {secretName, key}
(docs/secrets.md, kcc.secretRef), never in the manifest. */ -}}
{{- define "netlbapp.sslCertificates" -}}
{{- $v := .Values }}
{{- range $k := keys (($v.ssl_certificates | default dict).create_configs | default dict) | sortAlpha }}
{{- $s := index $v.ssl_certificates.create_configs $k }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSSLCertificate
metadata:
  {{- include "kcc.metadata" (dict "root" $ "name" (include "kcc.name" (list $.Release.Name "cert" $k)) "id" (printf "ssl_certificate:%s" $k) "project" $v.project_id "abandon" (include "kcc.abandon" $)) | nindent 2 }}
spec:
  resourceID: {{ include "kcc.cloudName" (list $ "SSL certificate" ($s.name | default (printf "%s-%s" $v.name $k)) 63 "letter") | quote }}
  location: {{ include "netlbapp.location" $ | quote }}
  certificate:
    value: {{ trim $s.certificate | quote }}
  privateKey: {{ include "kcc.secretRef" (list $ (printf "ssl_certificates.create_configs.%s.private_key" $k) $s.private_key) }}
{{- end }}
{{- end -}}

{{- /* netlbapp.proxyCertificates: the HTTPS proxy's sslCertificates: the
given IDs, then the chart's certificates (by Kubernetes name), then the
managed ones (by ID: a reference by name is to a ComputeSSLCertificate). */ -}}
{{- define "netlbapp.proxyCertificates" -}}
{{- $v := .Values }}
{{- $sc := $v.ssl_certificates | default dict }}
{{- range $sc.certificate_ids | default list }}
- external: {{ include "netlbapp.link" . | quote }}
{{- end }}
{{- range $k := keys ($sc.create_configs | default dict) | sortAlpha }}
- name: {{ include "kcc.name" (list $.Release.Name "cert" $k) }}
{{- end }}
{{- range $k := keys ($sc.managed_configs | default dict) | sortAlpha }}
{{- $m := index $sc.managed_configs $k }}
- external: {{ printf "projects/%s/global/sslCertificates/%s" $v.project_id (include "kcc.cloudName" (list $ "managed SSL certificate" ($m.name | default (printf "%s-%s" $v.name $k)) 63 "letter")) | quote }}
{{- end }}
{{- end -}}
