{{- define "agentclash.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "agentclash.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "agentclash.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "agentclash.labels" -}}
app.kubernetes.io/name: {{ include "agentclash.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "agentclash.selectorLabels" -}}
app.kubernetes.io/name: {{ include "agentclash.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "agentclash.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "agentclash.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "agentclash.apiImage" -}}
{{- $repo := .Values.apiServer.image.repository | default (printf "%s/%s-api-server" .Values.image.registry .Values.image.repository) -}}
{{- printf "%s:%s" $repo .Values.image.tag -}}
{{- end -}}

{{- define "agentclash.workerImage" -}}
{{- $repo := .Values.workers.image.repository | default (printf "%s/%s-worker" .Values.image.registry .Values.image.repository) -}}
{{- printf "%s:%s" $repo .Values.image.tag -}}
{{- end -}}

{{- define "agentclash.secretName" -}}
{{- if .Values.secrets.existingSecret -}}
{{- .Values.secrets.existingSecret -}}
{{- else -}}
{{- printf "%s-secrets" (include "agentclash.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "agentclash.temporalEndpoint" -}}
{{- if .Values.keda.temporalEndpoint -}}
{{- .Values.keda.temporalEndpoint -}}
{{- else -}}
{{- .Values.external.temporalAddress -}}
{{- end -}}
{{- end -}}
