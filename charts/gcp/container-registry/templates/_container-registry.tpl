{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0

containerRegistry.repository: the gcr.io repository for fabric's location,
as JSON {name, location}. Google fixes both by the Container Registry host.
*/ -}}
{{- define "containerRegistry.repository" -}}
{{- $l := upper (toString (.Values.location | default "")) -}}
{{- $hosts := dict "" (dict "name" "gcr.io" "location" "us") "US" (dict "name" "us.gcr.io" "location" "us") "EU" (dict "name" "eu.gcr.io" "location" "europe") "ASIA" (dict "name" "asia.gcr.io" "location" "asia") -}}
{{- if not (hasKey $hosts $l) -}}
{{- fail (printf "container-registry: location %q is not a Container Registry location: US, EU, ASIA or empty" .Values.location) -}}
{{- end -}}
{{- index $hosts $l | toJson -}}
{{- end -}}
