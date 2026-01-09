package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
)

type span struct {
	spanID       string
	parentSpanID string
	service      string
	operation    string
	duration     int64
	statusCode   int
	attributes   map[string]string
	startTime    int64
}

func SummarizeTrace(
	frames []interface{},
	traceID string,
	datasource string,
) (*TracesEvidence, error) {
	if len(frames) == 0 {
		return &TracesEvidence{
			TraceID:           traceID,
			TopSlowestSpans:   []SlowSpan{},
			CriticalPath:      []string{},
			NotableAttributes: make(map[string]string),
		}, nil
	}

	evidence := &TracesEvidence{
		TraceID:           traceID,
		TopSlowestSpans:   []SlowSpan{},
		CriticalPath:      []string{},
		NotableAttributes: make(map[string]string),
	}

	spans := []span{}

	for _, frame := range frames {
		frameBytes, err := json.Marshal(frame)
		if err != nil {
			continue
		}

		var frameData map[string]interface{}
		if err := json.Unmarshal(frameBytes, &frameData); err != nil {
			continue
		}

		schema, ok := frameData["schema"].(map[string]interface{})
		if !ok {
			continue
		}

		fields, ok := schema["fields"].([]interface{})
		if !ok {
			continue
		}

		fieldIndices := make(map[string]int)
		for i, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			fieldName, _ := fieldMap["name"].(string)
			fieldIndices[fieldName] = i
		}

		data, ok := frameData["data"].(map[string]interface{})
		if !ok {
			continue
		}

		values, ok := data["values"].([]interface{})
		if !ok || len(values) == 0 {
			continue
		}

		numRows := 0
		if len(values) > 0 {
			if firstCol, ok := values[0].([]interface{}); ok {
				numRows = len(firstCol)
			}
		}

		for row := 0; row < numRows; row++ {
			s := span{
				attributes: make(map[string]string),
			}

			for fieldName, idx := range fieldIndices {
				if idx >= len(values) {
					continue
				}

				column, ok := values[idx].([]interface{})
				if !ok || row >= len(column) {
					continue
				}

				value := column[row]

				switch fieldName {
				case "spanID", "span_id":
					if v, ok := value.(string); ok {
						s.spanID = v
					}
				case "parentSpanID", "parent_span_id":
					if v, ok := value.(string); ok {
						s.parentSpanID = v
					}
				case "serviceName", "service_name", "service":
					if v, ok := value.(string); ok {
						s.service = v
					}
				case "operationName", "operation_name", "operation":
					if v, ok := value.(string); ok {
						s.operation = v
					}
				case "duration":
					if v, ok := value.(float64); ok {
						s.duration = int64(v)
					} else if v, ok := value.(int64); ok {
						s.duration = v
					}
				case "statusCode", "status_code":
					if v, ok := value.(float64); ok {
						s.statusCode = int(v)
					}
				case "startTime", "start_time":
					if v, ok := value.(float64); ok {
						s.startTime = int64(v)
					}
				}
			}

			if s.spanID != "" {
				spans = append(spans, s)
			}
		}
	}

	if len(spans) == 0 {
		return evidence, nil
	}

	evidence.SpanCount = len(spans)

	var rootSpan *span
	for i := range spans {
		if spans[i].parentSpanID == "" {
			rootSpan = &spans[i]
			break
		}
	}

	if rootSpan != nil {
		evidence.RootService = rootSpan.service
		evidence.RootOperation = rootSpan.operation
		evidence.TotalDuration = rootSpan.duration
	}

	errorCount := 0
	for _, s := range spans {
		if s.statusCode != 0 && s.statusCode != 1 {
			errorCount++
		}
	}
	evidence.ErrorSpanCount = errorCount

	spansCopy := make([]span, len(spans))
	copy(spansCopy, spans)
	sort.Slice(spansCopy, func(i, j int) bool {
		return spansCopy[i].duration > spansCopy[j].duration
	})

	topN := 5
	if len(spansCopy) < topN {
		topN = len(spansCopy)
	}

	evidence.TopSlowestSpans = make([]SlowSpan, topN)
	for i := 0; i < topN; i++ {
		evidence.TopSlowestSpans[i] = SlowSpan{
			Service:   spansCopy[i].service,
			Operation: spansCopy[i].operation,
			Duration:  spansCopy[i].duration,
		}
	}

	evidence.CriticalPath = calculateCriticalPath(spans)

	evidence.NotableAttributes = extractNotableAttributes(spans)

	return evidence, nil
}

func calculateCriticalPath(spans []span) []string {
	if len(spans) == 0 {
		return []string{}
	}

	spanMap := make(map[string]*span)
	for i := range spans {
		spanMap[spans[i].spanID] = &spans[i]
	}

	var rootSpan *span
	for i := range spans {
		if spans[i].parentSpanID == "" {
			rootSpan = &spans[i]
			break
		}
	}

	if rootSpan == nil {
		return []string{}
	}

	type pathNode struct {
		span     *span
		duration int64
	}

	var findLongestPath func(*span) ([]pathNode, int64)
	findLongestPath = func(current *span) ([]pathNode, int64) {
		children := []*span{}
		for _, s := range spans {
			if s.parentSpanID == current.spanID {
				children = append(children, spanMap[s.spanID])
			}
		}

		if len(children) == 0 {
			return []pathNode{{span: current, duration: current.duration}}, current.duration
		}

		var longestChildPath []pathNode
		var longestChildDuration int64

		for _, child := range children {
			childPath, childDuration := findLongestPath(child)
			if childDuration > longestChildDuration {
				longestChildPath = childPath
				longestChildDuration = childDuration
			}
		}

		path := append([]pathNode{{span: current, duration: current.duration}}, longestChildPath...)
		return path, current.duration + longestChildDuration
	}

	path, _ := findLongestPath(rootSpan)

	criticalPath := make([]string, len(path))
	for i, node := range path {
		criticalPath[i] = fmt.Sprintf("%s:%s", node.span.service, node.span.operation)
	}

	return criticalPath
}

func extractNotableAttributes(spans []span) map[string]string {
	attributes := make(map[string]string)

	for _, s := range spans {
		for key, value := range s.attributes {
			if key == "error" || key == "error.message" || key == "exception.message" {
				attributes[key] = value
			}
			if key == "service.version" || key == "version" {
				attributes[s.service+"_version"] = value
			}
		}
	}

	return attributes
}

func BuildTracesEvidencePack(
	grafanaResp *GrafanaQueryResponse,
	traceID string,
	datasource string,
) (*EvidencePack, error) {
	result, ok := grafanaResp.Results["A"]
	if !ok {
		return nil, fmt.Errorf("no results for refId A")
	}

	if result.Error != "" {
		return nil, fmt.Errorf("query error: %s", result.Error)
	}

	traces, err := SummarizeTrace(result.Frames, traceID, datasource)
	if err != nil {
		return nil, err
	}

	quality := QualityGood
	if traces.SpanCount == 0 {
		quality = QualityNoData
	}

	return &EvidencePack{
		Type:       "traces",
		Datasource: datasource,
		TimeRange:  TimeRange{},
		Query:      traceID,
		Traces:     traces,
		Quality:    quality,
	}, nil
}
