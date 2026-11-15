{{- /*
Copyright 2026 High Five Labs
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{/* dataformRepo.name: the repository's name: names.repository in full,
else name with the namePrefix (docs/design.md, "Names, labels and tags"). */}}
{{- define "dataformRepo.name" -}}
{{- include "kcc.resourceName" (list . "repository" "repository" .Values.name 63) -}}
{{- end -}}

{{/* dataformRepo.id: the repository's full name, the resource IAM is on. */}}
{{- define "dataformRepo.id" -}}
{{- printf "projects/%s/locations/%s/repositories/%s" .Values.project_id .Values.region (include "dataformRepo.name" .) -}}
{{- end -}}
