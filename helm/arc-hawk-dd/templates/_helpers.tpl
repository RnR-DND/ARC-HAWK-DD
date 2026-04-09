{{/*
Expand the name of the chart.
*/}}
{{- define "arc-hawk-dd.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "arc-hawk-dd.fullname" -}}
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
Chart label.
*/}}
{{- define "arc-hawk-dd.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "arc-hawk-dd.labels" -}}
helm.sh/chart: {{ include "arc-hawk-dd.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Selector labels for a given component.
Usage: include "arc-hawk-dd.selectorLabels" (dict "root" . "component" "backend")
*/}}
{{- define "arc-hawk-dd.selectorLabels" -}}
app.kubernetes.io/name: {{ include "arc-hawk-dd.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Image helper: prepends global registry if set.
*/}}
{{- define "arc-hawk-dd.image" -}}
{{- $reg := .root.Values.global.imageRegistry -}}
{{- if $reg -}}
{{- printf "%s/%s:%s" (trimSuffix "/" $reg) .repo .tag -}}
{{- else -}}
{{- printf "%s:%s" .repo .tag -}}
{{- end -}}
{{- end }}
