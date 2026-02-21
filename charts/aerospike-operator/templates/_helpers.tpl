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
Common labels
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
Selector labels
*/}}
{{- define "aerospike-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aerospike-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "aerospike-operator.serviceAccountName" -}}
{{- include "aerospike-operator.fullname" . }}
{{- end }}

{{/*
Webhook service name
*/}}
{{- define "aerospike-operator.webhookServiceName" -}}
{{- include "aerospike-operator.fullname" . }}-webhook
{{- end }}

{{/*
Metrics service name
*/}}
{{- define "aerospike-operator.metricsServiceName" -}}
{{- include "aerospike-operator.fullname" . }}-metrics
{{- end }}

{{/*
Cert-manager issuer name
*/}}
{{- define "aerospike-operator.issuerName" -}}
{{- include "aerospike-operator.fullname" . }}-selfsigned-issuer
{{- end }}

{{/*
Cert-manager certificate secret name
*/}}
{{- define "aerospike-operator.certSecretName" -}}
{{- include "aerospike-operator.fullname" . }}-webhook-cert
{{- end }}

{{/*
Cert-manager certificate name
*/}}
{{- define "aerospike-operator.certName" -}}
{{- include "aerospike-operator.fullname" . }}-serving-cert
{{- end }}

{{/*
Container image
*/}}
{{- define "aerospike-operator.image" -}}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end }}
