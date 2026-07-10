package stamp

import (
	"github.com/zalegrala/chartwright/interchange"
	"sigs.k8s.io/yaml"
)

func chartYAML(c interchange.Chart) ([]byte, error) {
	doc := map[string]interface{}{
		"apiVersion": "v2",
		"type":       "application",
		"name":       c.Name,
		"version":    c.Version,
	}
	if c.AppVersion != "" {
		doc["appVersion"] = c.AppVersion
	}
	if c.Description != "" {
		doc["description"] = c.Description
	}
	if c.KubeVersion != "" {
		doc["kubeVersion"] = c.KubeVersion
	}
	return yaml.Marshal(doc)
}

// helpersTpl is a standard set of name/label helpers. It references .Chart.Name
// at install time, so it needs no per-chart parameterization (keeps output stable).
const helpersTpl = `{{- define "chart.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "chart.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "chart.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
`
