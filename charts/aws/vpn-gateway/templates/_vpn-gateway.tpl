{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

vpn.secretRef: a reference to a Kubernetes Secret, {secretName, key}
(docs/secrets.md), checked, as JSON {name, key}: VPNConnection's tunnel
option preSharedKey (ACK's SecretKeyReference). A string, the secret
itself, is refused: it would be in the rendered manifest.
  {{ include "vpn.secretRef" (list $ "tunnel1_preshared_key" .Values.tunnel1_preshared_key) }}
*/ -}}
{{- define "vpn.secretRef" -}}
{{- $root := index . 0 -}}{{- $path := index . 1 -}}{{- $v := index . 2 -}}
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
