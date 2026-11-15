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

{{- /*
containerRegistry.iam: fabric's iam with its Cloud Storage roles translated
to Artifact Registry roles (the builder, 2026-11-15, CRREG-1; docs/design.md,
decision 13), as JSON {iam, notes}. Fabric bound iam on the registry's Cloud
Storage bucket; a gcr.io repository on Artifact Registry takes Artifact
Registry roles only. Exactly five storage roles translate, by our own table
(Google gives none); two roles that translate to one Artifact Registry role,
or to one also listed as is, share one binding. Every other roles/storage.*
role is refused. Each converted role gives one note, which iam.yaml writes
into the render and NOTES.txt prints on install.
*/ -}}
{{- define "containerRegistry.iam" -}}
{{- $map := dict "roles/storage.objectViewer" "roles/artifactregistry.reader" "roles/storage.legacyBucketReader" "roles/artifactregistry.reader" "roles/storage.legacyBucketWriter" "roles/artifactregistry.writer" "roles/storage.legacyBucketOwner" "roles/artifactregistry.repoAdmin" "roles/storage.admin" "roles/artifactregistry.repoAdmin" -}}
{{- $iam := dict -}}
{{- $notes := list -}}
{{- $v := .Values.iam | default dict -}}
{{- range $role := keys $v | sortAlpha -}}
{{- $to := $role -}}
{{- if hasKey $map $role -}}
{{- $to = index $map $role -}}
{{- $notes = append $notes (printf "container-registry: iam role %s was converted to %s. gcr.io repositories are legacy, supported for organizations still on gcr.io; new setups should use a native Artifact Registry repository with Artifact Registry roles directly." $role $to) -}}
{{- else if hasPrefix "roles/storage." $role -}}
{{- include "kcc.unsupported" (list $ (printf "iam.%s" $role) "set" "the registry is an Artifact Registry repository, not a Cloud Storage bucket: grant an Artifact Registry role instead (roles/artifactregistry.reader to pull, roles/artifactregistry.writer to push, roles/artifactregistry.repoAdmin to also delete)") -}}
{{- end -}}
{{- $_ := set $iam $to (concat (get $iam $to | default list) (index $v $role | default list) | uniq | sortAlpha) -}}
{{- end -}}
{{- toJson (dict "iam" $iam "notes" $notes) -}}
{{- end -}}
