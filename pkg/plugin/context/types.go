package context

import "time"

type ObservabilityContext struct {
	Metrics *MetricsContext `json:"metrics,omitempty"`
	Logs    *LogsContext    `json:"logs,omitempty"`
	Traces  *TracesContext  `json:"traces,omitempty"`

	LastUpdated time.Time `json:"lastUpdated"`
}

type MetricsContext struct {
	MetricNames []string            `json:"metricNames"`
	Labels      []string            `json:"labels"`
	LabelValues map[string][]string `json:"labelValues"` 
	SampleCount int                 `json:"sampleCount"`
}

type LogsContext struct {
	Streams     []string            `json:"streams"`
	Labels      []string            `json:"labels"`
	LabelValues map[string][]string `json:"labelValues"`
	SampleCount int                 `json:"sampleCount"`
}

type TracesContext struct {
	Services    []string `json:"services"`
	Operations  []string `json:"operations"`
	SpanNames   []string `json:"spanNames"`
	SampleCount int      `json:"sampleCount"`
}

func (oc *ObservabilityContext) BuildPrompt() string {
	if oc == nil {
		return ""
	}

	prompt := "You are helping with Grafana observability. Here is the current system context:\n\n"

	if oc.Metrics != nil && len(oc.Metrics.MetricNames) > 0 {
		prompt += "=== METRICS ===\n"
		prompt += "Available metric names (sample):\n"

		maxMetrics := 50
		for i, metric := range oc.Metrics.MetricNames {
			if i >= maxMetrics {
				prompt += "... and more\n"
				break
			}
			prompt += "- " + metric + "\n"
		}

		if len(oc.Metrics.Labels) > 0 {
			prompt += "\nCommon labels:\n"
			maxLabels := 30
			for i, label := range oc.Metrics.Labels {
				if i >= maxLabels {
					prompt += "... and more\n"
					break
				}
				prompt += "- " + label

				if values, ok := oc.Metrics.LabelValues[label]; ok && len(values) > 0 {
					prompt += " (examples: "
					maxVals := 3
					for j, val := range values {
						if j >= maxVals {
							prompt += "..."
							break
						}
						if j > 0 {
							prompt += ", "
						}
						prompt += val
					}
					prompt += ")"
				}
				prompt += "\n"
			}
		}
		prompt += "\n"
	}

	if oc.Logs != nil && len(oc.Logs.Streams) > 0 {
		prompt += "=== LOGS ===\n"
		prompt += "Available log streams (sample):\n"

		maxStreams := 30
		for i, stream := range oc.Logs.Streams {
			if i >= maxStreams {
				prompt += "... and more\n"
				break
			}
			prompt += "- " + stream + "\n"
		}

		if len(oc.Logs.Labels) > 0 {
			prompt += "\nLog labels:\n"
			maxLabels := 20
			for i, label := range oc.Logs.Labels {
				if i >= maxLabels {
					prompt += "... and more\n"
					break
				}
				prompt += "- " + label + "\n"
			}
		}
		prompt += "\n"
	}

	if oc.Traces != nil && len(oc.Traces.Services) > 0 {
		prompt += "=== TRACES ===\n"

		if len(oc.Traces.Services) > 0 {
			prompt += "Services:\n"
			maxServices := 30
			for i, service := range oc.Traces.Services {
				if i >= maxServices {
					prompt += "... and more\n"
					break
				}
				prompt += "- " + service + "\n"
			}
		}

		if len(oc.Traces.Operations) > 0 {
			prompt += "\nOperations (sample):\n"
			maxOps := 20
			for i, op := range oc.Traces.Operations {
				if i >= maxOps {
					prompt += "... and more\n"
					break
				}
				prompt += "- " + op + "\n"
			}
		}
		prompt += "\n"
	}

	prompt += "When helping with queries, alerts, or dashboards, reference these actual metrics, labels, and services from the system.\n"

	return prompt
}
