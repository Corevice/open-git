{{/*
Expand the name of the chart.
*/}}
{{- define "open-git.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "open-git.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "open-git.labels" -}}
helm.sh/chart: {{ include "open-git.chart" . }}
{{ include "open-git.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "open-git.selectorLabels" -}}
app.kubernetes.io/name: {{ include "open-git.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart label
*/}}
{{- define "open-git.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Validated Git repository mount path from values.
*/}}
{{- define "open-git.repositoriesMountPath" -}}
{{- $path := .Values.persistence.repositories.mountPath | default "/data/repositories" -}}
{{- if or (not (hasPrefix "/" $path)) (contains ".." $path) (contains "\n" $path) -}}
{{- fail (printf "invalid persistence.repositories.mountPath %q: must be an absolute path without .. or newlines" $path) -}}
{{- end -}}
{{- $path -}}
{{- end }}
