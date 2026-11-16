{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
Helpers for the Cloud Router and its route policies, as fabric's VPN and
interconnect modules (net-vpn-ha, net-vpn-dynamic, net-vlan-attachment,
net-ipsec-over-interconnect, v60.0.0) write them: the same in each of
those charts.
*/ -}}

{{/*
router.bgp: a ComputeRouter's bgp from fabric's router_config (asn,
custom_advertise, keepalive). A CUSTOM advertise mode when
custom_advertise is set, as fabric's.
*/}}
{{- define "router.bgp" -}}
{{- $rc := . -}}
asn: {{ int64 $rc.asn }}
advertiseMode: {{ ternary "CUSTOM" "DEFAULT" (not (empty $rc.custom_advertise)) }}
{{- with $rc.custom_advertise }}
{{- if .all_subnets }}
advertisedGroups: [ALL_SUBNETS]
{{- end }}
{{- with .ip_ranges }}
advertisedIpRanges:
  {{- range $range, $desc := . }}
  - range: {{ $range | quote }}
    description: {{ $desc | quote }}
  {{- end }}
{{- end }}
{{- end }}
{{- if not (kindIs "invalid" $rc.keepalive) }}
keepaliveInterval: {{ int64 $rc.keepalive }}
{{- end }}
{{- end -}}

{{/*
router.expr: a route policy term's match or action ({expression, title,
description, location}).
*/}}
{{- define "router.expr" -}}
expression: {{ .expression | quote }}
{{- with .title }}
title: {{ . | quote }}
{{- end }}
{{- with .description }}
description: {{ . | quote }}
{{- end }}
{{- with .location }}
location: {{ . | quote }}
{{- end }}
{{- end -}}

{{/*
router.policies: one config-connector-extensions ComputeRouterRoutePolicy
(v1beta1) per entry of router_config.route_policies,
named by its key, on the router the chart makes or the one router_config
names. Checks fabric's validation that a policy's term priorities are
unique (the schema checks the type and the priority's range).
  root      $
  policies  router_config.route_policies
  router    {name: <ComputeRouter object>} or {external: projects/.../routers/<name>}
  abandon   kcc.abandon's result
*/}}
{{- define "router.policies" -}}
{{- $root := .root -}}{{- $v := $root.Values -}}
{{- range $k, $p := .policies }}
{{- $seen := dict }}
{{- range $t := $p.terms }}
{{- if hasKey $seen (toString $t.priority) }}
{{- fail (printf "%s: Route policy term priority must be unique within the policy." $root.Chart.Name) }}
{{- end }}
{{- $_ := set $seen (toString $t.priority) true }}
{{- end }}
{{- end }}
{{- $router := .router }}
{{- $abandon := .abandon }}
{{- range $key, $p := .policies }}
---
apiVersion: compute.gcp.hfl.dev/v1beta1
kind: ComputeRouterRoutePolicy
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (list $root.Release.Name "route-policy" $key)) "id" (printf "route_policy:%s" $key) "project" $v.project_id "abandon" $abandon) | nindent 2 }}
spec:
  resourceID: {{ include "kcc.cloudName" (list $root "route policy" $key 63 "letter") | quote }}
  projectRef:
    external: {{ $v.project_id | quote }}
  region: {{ $v.region | quote }}
  routerRef:
    {{- toYaml $router | nindent 4 }}
  type: {{ printf "ROUTE_POLICY_TYPE_%s" $p.type }}
  terms:
    {{- range $t := $p.terms }}
    - priority: {{ int64 $t.priority }}
      match:
        {{- include "router.expr" $t.match | nindent 8 }}
      {{- with $t.actions }}
      actions:
        {{- range $a := . }}
        - {{ include "router.expr" $a | nindent 10 | trim }}
        {{- end }}
      {{- end }}
    {{- end }}
{{- end }}
{{- end -}}

{{/*
router.peerRouter: the router a BGP peer is on, as ComputeRouterPeer's
routerRef takes it: the chart's router object by name, or another router
by its name alone (the peer's routerRef.external is a router's name, where
the interface's and tunnel's are projects/<p>/regions/<r>/routers/<name>).
  router  {name: <ComputeRouter object>} or {external: projects/.../routers/<name>}
  other   a tunnel's router (fabric's tunnels[].router), or null
*/}}
{{- define "router.peerRouter" -}}
{{- if .other -}}
{{- toJson (dict "external" (last (splitList "/" .other))) -}}
{{- else if .router.name -}}
{{- toJson (dict "name" .router.name) -}}
{{- else -}}
{{- toJson (dict "external" (last (splitList "/" .router.external))) -}}
{{- end -}}
{{- end -}}

{{/*
router.tunnelBgp: what fabric makes for each tunnel of a BGP VPN, beside
the tunnel: google_compute_router_interface (a ComputeRouterInterface on
the tunnel, its ipRange the tunnel's bgp_session_range) and
google_compute_router_peer (a ComputeRouterPeer on that interface, from
the tunnel's bgp_peer). Refuses what ComputeRouterPeer has no field for:
md5_authentication_key, export_policies, import_policies and
custom_learned_ip_ranges.
  root        $
  key         the tunnel's key
  t           the tunnel
  ifaceName   the interface's cloud name
  peerName    the peer's cloud name
  tunnel      the ComputeVPNTunnel object's name
  router      the interface's router: {name: <object>} or {external: projects/.../routers/<name>}
  peerRouter  the peer's router (router.peerRouter's result, JSON)
  abandon     kcc.abandon's result
*/}}
{{- define "router.tunnelBgp" -}}
{{- $root := .root -}}{{- $v := $root.Values -}}{{- $key := .key -}}{{- $t := .t -}}
{{- $p := $t.bgp_peer | default dict -}}
{{- range $f := list "md5_authentication_key" "export_policies" "import_policies" "custom_learned_ip_ranges" }}
{{- include "kcc.unsupported" (list $root (printf "tunnels.%s.bgp_peer.%s" $key $f) (index $p $f) "ComputeRouterPeer has no md5AuthenticationKey, importPolicies, exportPolicies or customLearnedIpRanges") }}
{{- end }}
{{- $iface := include "kcc.name" (list $root.Release.Name "interface" $key) }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeRouterInterface
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" $iface "id" (printf "router_interface:%s" $key) "project" $v.project_id "abandon" .abandon) | nindent 2 }}
spec:
  resourceID: {{ .ifaceName | quote }}
  region: {{ $v.region | quote }}
  routerRef:
    {{- toYaml .router | nindent 4 }}
  {{- with $t.bgp_session_range }}
  ipRange: {{ . | quote }}
  {{- end }}
  vpnTunnelRef:
    name: {{ .tunnel }}
---
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeRouterPeer
metadata:
  {{- include "kcc.metadata" (dict "root" $root "name" (include "kcc.name" (list $root.Release.Name "bgp-peer" $key)) "id" (printf "bgp_peer:%s" $key) "project" $v.project_id "abandon" .abandon) | nindent 2 }}
spec:
  resourceID: {{ .peerName | quote }}
  region: {{ $v.region | quote }}
  routerRef: {{ .peerRouter }}
  routerInterfaceRef:
    name: {{ $iface }}
  peerIpAddress: {{ $p.address | quote }}
  peerAsn: {{ int64 $p.asn }}
  advertisedRoutePriority: {{ ternary 1000 $p.route_priority (kindIs "invalid" $p.route_priority) | int64 }}
  {{- $ca := $p.custom_advertise }}
  advertiseMode: {{ ternary "DEFAULT" "CUSTOM" (kindIs "invalid" $ca) }}
  {{- with $ca }}
  {{- $groups := list }}
  {{- if .all_subnets }}{{ $groups = append $groups "ALL_SUBNETS" }}{{ end }}
  {{- if .all_vpc_subnets }}{{ $groups = append $groups "ALL_VPC_SUBNETS" }}{{ end }}
  {{- if .all_peer_vpc_subnets }}{{ $groups = append $groups "ALL_PEER_VPC_SUBNETS" }}{{ end }}
  {{- with $groups }}
  advertisedGroups: {{ toJson . }}
  {{- end }}
  {{- with .ip_ranges }}
  advertisedIpRanges:
    {{- range $range, $desc := . }}
    - range: {{ $range | quote }}
      description: {{ $desc | quote }}
    {{- end }}
  {{- end }}
  {{- end }}
  {{- with $p.ipv6 }}
  enableIpv6: true
  {{- with .nexthop_address }}
  ipv6NexthopAddress: {{ . | quote }}
  {{- end }}
  {{- with .peer_nexthop_address }}
  peerIpv6NexthopAddress: {{ . | quote }}
  {{- end }}
  {{- end }}
{{- end -}}
