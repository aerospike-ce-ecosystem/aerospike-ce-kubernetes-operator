{{/*
Expand the name of the chart.
*/}}
{{- define "acko.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "acko.fullname" -}}
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
{{- define "acko.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to all resources.
*/}}
{{- define "acko.labels" -}}
helm.sh/chart: {{ include "acko.chart" . }}
{{ include "acko.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels used for pod selection.
*/}}
{{- define "acko.selectorLabels" -}}
app.kubernetes.io/name: {{ include "acko.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use.
*/}}
{{- define "acko.serviceAccountName" -}}
{{- include "acko.fullname" . }}
{{- end }}

{{/*
Webhook service name.
*/}}
{{- define "acko.webhookServiceName" -}}
{{- include "acko.fullname" . }}-webhook
{{- end }}

{{/*
Metrics service name.
*/}}
{{- define "acko.metricsServiceName" -}}
{{- include "acko.fullname" . }}-metrics
{{- end }}

{{/*
Cert-manager issuer name.
*/}}
{{- define "acko.issuerName" -}}
{{- include "acko.fullname" . }}-selfsigned-issuer
{{- end }}

{{/*
Cert-manager certificate secret name.
*/}}
{{- define "acko.certSecretName" -}}
{{- if and .Values.certManager.enabled .Values.webhook.enabled }}
{{- include "acko.fullname" . }}-webhook-cert
{{- else if .Values.webhookTlsSecret }}
{{- .Values.webhookTlsSecret }}
{{- else }}
{{- include "acko.fullname" . }}-webhook-cert
{{- end }}
{{- end }}

{{/*
Cert-manager certificate name.
*/}}
{{- define "acko.certName" -}}
{{- include "acko.fullname" . }}-serving-cert
{{- end }}

{{/*
Container image with tag.
*/}}
{{- define "acko.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{/*
Pod labels combining selector labels with user-defined pod labels.
*/}}
{{- define "acko.podLabels" -}}
{{ include "acko.selectorLabels" . }}
{{- with .Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Namespace for the release.
*/}}
{{- define "acko.namespace" -}}
{{- .Release.Namespace }}
{{- end }}

{{/*
=============================================================================
UI (Aerospike Cluster Manager) helpers
=============================================================================
*/}}

{{/*
UI component name (constant).
*/}}
{{- define "acko.ui.name" -}}
aerospike-cluster-manager
{{- end }}

{{/*
UI fully qualified name.
*/}}
{{- define "acko.ui.fullname" -}}
{{- include "acko.fullname" . }}-ui
{{- end }}

{{/*
UI common labels.
*/}}
{{- define "acko.ui.labels" -}}
helm.sh/chart: {{ include "acko.chart" . }}
{{ include "acko.ui.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: ui
{{- end }}

{{/*
UI selector labels.
*/}}
{{- define "acko.ui.selectorLabels" -}}
app.kubernetes.io/name: {{ include "acko.ui.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
UI service account name.
*/}}
{{- define "acko.ui.serviceAccountName" -}}
{{- if .Values.ui.serviceAccount.create }}
{{- include "acko.ui.fullname" . }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}

{{/*
UI container image with tag.
*/}}
{{- define "acko.ui.image" -}}
{{- printf "%s:%s" .Values.ui.image.repository (default .Chart.AppVersion .Values.ui.image.tag) }}
{{- end }}
