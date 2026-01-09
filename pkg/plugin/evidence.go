package plugin

type EvidencePack struct {
	Type       string           `json:"type"`
	Datasource string           `json:"datasource"`
	TimeRange  TimeRange        `json:"timeRange"`
	Query      string           `json:"query"`
	Metrics    *MetricsEvidence `json:"metrics,omitempty"`
	Logs       *LogsEvidence    `json:"logs,omitempty"`
	Traces     *TracesEvidence  `json:"traces,omitempty"`
	Quality    string           `json:"quality"`
}

type MetricsEvidence struct {
	Unit            string              `json:"unit"`
	SeriesCount     int                 `json:"seriesCount"`
	Current         float64             `json:"current"`
	Min             float64             `json:"min"`
	Max             float64             `json:"max"`
	Avg             float64             `json:"avg"`
	Trend           string              `json:"trend"`
	SlopePerHour    float64             `json:"slopePerHour"`
	Quality         string              `json:"quality"`
	TopContributors []LabelContributor  `json:"topContributors,omitempty"`
}

type LabelContributor struct {
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

type LogsEvidence struct {
	TotalCount      int64               `json:"totalCount"`
	Rate            float64             `json:"rate"`
	MaxRate         float64             `json:"maxRate"`
	Trend           string              `json:"trend"`
	TopLabels       map[string][]string `json:"topLabels"`
	NotableMessages []string            `json:"notableMessages"`
}

type TracesEvidence struct {
	TraceID           string            `json:"traceId"`
	RootService       string            `json:"rootService"`
	RootOperation     string            `json:"rootOperation"`
	TotalDuration     int64             `json:"totalDuration"`
	SpanCount         int               `json:"spanCount"`
	ErrorSpanCount    int               `json:"errorSpanCount"`
	TopSlowestSpans   []SlowSpan        `json:"topSlowestSpans"`
	CriticalPath      []string          `json:"criticalPath"`
	NotableAttributes map[string]string `json:"notableAttributes"`
}

type SlowSpan struct {
	Service   string `json:"service"`
	Operation string `json:"operation"`
	Duration  int64  `json:"duration"`
}

const (
	TrendIncreasing = "increasing"
	TrendDecreasing = "decreasing"
	TrendFlat       = "flat"
	TrendSpiky      = "spiky"
	TrendNoData     = "no_data"

	QualityGood         = "good"
	QualityGaps         = "gaps"
	QualityNoData       = "no_data"
	QualityInsufficient = "insufficient"
)
