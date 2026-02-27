{{/*
Expand the name of the chart.
*/}}
{{- define "aerospike-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "aerospike-operator.fullname" -}}
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
{{- define "aerospike-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to all resources.
*/}}
{{- define "aerospike-operator.labels" -}}
helm.sh/chart: {{ include "aerospike-operator.chart" . }}
{{ include "aerospike-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels used for pod selection.
*/}}
{{- define "aerospike-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aerospike-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use.
*/}}
{{- define "aerospike-operator.serviceAccountName" -}}
{{- include "aerospike-operator.fullname" . }}
{{- end }}

{{/*
Webhook service name.
*/}}
{{- define "aerospike-operator.webhookServiceName" -}}
{{- include "aerospike-operator.fullname" . }}-webhook
{{- end }}

{{/*
Metrics service name.
*/}}
{{- define "aerospike-operator.metricsServiceName" -}}
{{- include "aerospike-operator.fullname" . }}-metrics
{{- end }}

{{/*
Cert-manager issuer name.
*/}}
{{- define "aerospike-operator.issuerName" -}}
{{- include "aerospike-operator.fullname" . }}-selfsigned-issuer
{{- end }}

{{/*
Cert-manager certificate secret name.
*/}}
{{- define "aerospike-operator.certSecretName" -}}
{{- if and .Values.certManager.enabled .Values.webhook.enabled }}
{{- include "aerospike-operator.fullname" . }}-webhook-cert
{{- else if .Values.webhookTlsSecret }}
{{- .Values.webhookTlsSecret }}
{{- else }}
{{- include "aerospike-operator.fullname" . }}-webhook-cert
{{- end }}
{{- end }}

{{/*
Cert-manager certificate name.
*/}}
{{- define "aerospike-operator.certName" -}}
{{- include "aerospike-operator.fullname" . }}-serving-cert
{{- end }}

{{/*
Container image with tag.
*/}}
{{- define "aerospike-operator.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{/*
Pod labels combining selector labels with user-defined pod labels.
*/}}
{{- define "aerospike-operator.podLabels" -}}
{{ include "aerospike-operator.selectorLabels" . }}
{{- with .Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Namespace for the release.
*/}}
{{- define "aerospike-operator.namespace" -}}
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
{{- define "aerospike-operator.ui.name" -}}
aerospike-cluster-manager
{{- end }}

{{/*
UI fully qualified name.
*/}}
{{- define "aerospike-operator.ui.fullname" -}}
{{- include "aerospike-operator.fullname" . }}-ui
{{- end }}

{{/*
UI common labels.
*/}}
{{- define "aerospike-operator.ui.labels" -}}
helm.sh/chart: {{ include "aerospike-operator.chart" . }}
{{ include "aerospike-operator.ui.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: ui
{{- end }}

{{/*
UI selector labels.
*/}}
{{- define "aerospike-operator.ui.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aerospike-operator.ui.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
UI service account name.
*/}}
{{- define "aerospike-operator.ui.serviceAccountName" -}}
{{- if .Values.ui.serviceAccount.create }}
{{- include "aerospike-operator.ui.fullname" . }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}

{{/*
UI container image with tag.
*/}}
{{- define "aerospike-operator.ui.image" -}}
{{- printf "%s:%s" .Values.ui.image.repository (default "latest" .Values.ui.image.tag) }}
{{- end }}
