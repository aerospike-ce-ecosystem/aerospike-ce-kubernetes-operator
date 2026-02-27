{{/*
Expand the name of the chart.
*/}}
{{- define "aerospike-cluster-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "aerospike-cluster-manager.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "aerospike-cluster-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "aerospike-cluster-manager.labels" -}}
helm.sh/chart: {{ include "aerospike-cluster-manager.chart" . }}
{{ include "aerospike-cluster-manager.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "aerospike-cluster-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aerospike-cluster-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name.
*/}}
{{- define "aerospike-cluster-manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- include "aerospike-cluster-manager.fullname" . }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}

{{/*
Container image with tag.
*/}}
{{- define "aerospike-cluster-manager.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}
