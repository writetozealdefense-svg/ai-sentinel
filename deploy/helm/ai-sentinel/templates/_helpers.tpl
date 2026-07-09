{{/*
Common labels for a component.
Call with: (dict "root" $ "component" "<name>")
*/}}
{{- define "ai-sentinel.labels" -}}
app.kubernetes.io/name: {{ .component }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/part-of: ai-sentinel
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .root.Chart.Name .root.Chart.Version }}
{{- end }}

{{/*
Selector labels (stable subset — never add chart/version here).
Call with: (dict "root" $ "component" "<name>")
*/}}
{{- define "ai-sentinel.selectorLabels" -}}
app.kubernetes.io/name: {{ .component }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
{{- end }}

{{/*
Pod Security Standards "restricted"-compliant container securityContext.
Call with: (dict "uid" <numeric uid> "readOnlyRootFilesystem" <bool>)
Every field required by the restricted profile is set here; per-image uid and
read-only-root decisions are documented at each call site.
*/}}
{{- define "ai-sentinel.containerSecurityContext" -}}
allowPrivilegeEscalation: false
runAsNonRoot: true
runAsUser: {{ .uid }}
runAsGroup: {{ .gid | default .uid }}
readOnlyRootFilesystem: {{ .readOnlyRootFilesystem }}
capabilities:
  drop:
    - ALL
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{/*
Pod-level securityContext (restricted profile). Optional fsGroup for images
whose emptyDir volumes must be group-writable by the runtime uid.
Call with: (dict) or (dict "fsGroup" <gid>)
*/}}
{{- define "ai-sentinel.podSecurityContext" -}}
runAsNonRoot: true
seccompProfile:
  type: RuntimeDefault
{{- with .fsGroup }}
fsGroup: {{ . }}
fsGroupChangePolicy: OnRootMismatch
{{- end }}
{{- end }}
