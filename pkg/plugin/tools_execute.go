package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// executePromQL executes a PromQL query and returns structured analytics
func (a *App) executePromQL(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	// Extract parameters
	datasourceUid, ok := args["datasourceUid"].(string)
	if !ok || datasourceUid == "" {
		return "", fmt.Errorf("datasourceUid is required")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	from, ok := args["from"].(string)
	if !ok || from == "" {
		from = "now-15m"
	}

	to, ok := args["to"].(string)
	if !ok || to == "" {
		to = "now"
	}

	// Step is optional - Grafana will calculate a reasonable step if not provided
	var step string
	if stepVal, ok := args["step"].(string); ok {
		step = stepVal
	}

	// Apply OTel enforcement if enabled
	if a.settings != nil && a.settings.OtelEnforcement.Enabled {
		// Extract service name and environment from args
		serviceName, _ := args["serviceName"].(string)
		environmentName, _ := args["environmentName"].(string)

		scope := &OtelScope{
			ServiceName:     serviceName,
			EnvironmentName: environmentName,
			Source:          "tool_args",
		}

		// Apply defaults
		a.applyOtelScopeDefaults(scope)

		// Validate scope
		if err := a.validateOtelScope(scope); err != nil {
			return "", fmt.Errorf("OTel validation failed: %w", err)
		}

		// Inject scope into query
		labelFormat := a.otelRegistry.GetFormat(datasourceUid)
		if labelFormat == nil {
			labelFormat = a.DiscoverOTelLabels(ctx, datasourceUid, DatasourcePrometheus)
			a.otelRegistry.SetFormat(datasourceUid, labelFormat)
		}

		labels := BuildOTelLabels(labelFormat, scope.ServiceName, scope.EnvironmentName)
		query = injectLabelsIntoQuery(query, labels)

		backend.Logger.Debug("OTel scope injected into PromQL query",
			"original", args["query"],
			"injected", query,
			"scope", scope,
		)
	}

	// Note: Datasource type validation is skipped here because it requires an HTTP request
	// for authentication. If the datasource type doesn't match the query type,
	// Grafana's query API will return an appropriate error.

	// Build query payload
	queryPayload := QueryPayload{
		Expr:          query,
		IntervalMs:    15000, // 15 seconds default
		MaxDataPoints: 1000,
		Format:        "time_series",
	}

	// If step provided, calculate intervalMs from it
	if step != "" {
		if intervalMs, err := parseDurationToMs(step); err == nil {
			queryPayload.IntervalMs = intervalMs
		}
	}

	// Build time range
	timeRange := TimeRange{
		From: from,
		To:   to,
	}

	// Execute query via Grafana query client
	if a.grafanaQueryClient == nil {
		return "", fmt.Errorf("Grafana query client not initialized")
	}

	response, err := a.grafanaQueryClient.ExecuteQuery(ctx, user, datasourceUid, "prometheus", queryPayload, timeRange)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %w", err)
	}

	// Check for errors in response
	if result, ok := response.Results["A"]; ok && result.Error != "" {
		return "", fmt.Errorf("query error: %s", result.Error)
	}

	// Format response as structured JSON with analytics
	result := formatPromQLResult(response, query)

	// Marshal to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// executeLogQL executes a LogQL query and returns log analysis
func (a *App) executeLogQL(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	// Extract parameters
	datasourceUid, ok := args["datasourceUid"].(string)
	if !ok || datasourceUid == "" {
		return "", fmt.Errorf("datasourceUid is required")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	from, ok := args["from"].(string)
	if !ok || from == "" {
		from = "now-15m"
	}

	to, ok := args["to"].(string)
	if !ok || to == "" {
		to = "now"
	}

	// Limit is optional - default to 1000
	limit := 1000
	if limitVal, ok := args["limit"].(float64); ok {
		limit = int(limitVal)
	}
	// Cap at 5000
	if limit > 5000 {
		limit = 5000
	}

	// Apply OTel enforcement if enabled
	if a.settings != nil && a.settings.OtelEnforcement.Enabled {
		// Extract service name and environment from args
		serviceName, _ := args["serviceName"].(string)
		environmentName, _ := args["environmentName"].(string)

		scope := &OtelScope{
			ServiceName:     serviceName,
			EnvironmentName: environmentName,
			Source:          "tool_args",
		}

		// Apply defaults
		a.applyOtelScopeDefaults(scope)

		// Validate scope
		if err := a.validateOtelScope(scope); err != nil {
			return "", fmt.Errorf("OTel validation failed: %w", err)
		}

		// Inject scope into query
		labelFormat := a.otelRegistry.GetFormat(datasourceUid)
		if labelFormat == nil {
			labelFormat = a.DiscoverOTelLabels(ctx, datasourceUid, DatasourceLoki)
			a.otelRegistry.SetFormat(datasourceUid, labelFormat)
		}

		labels := BuildOTelLabels(labelFormat, scope.ServiceName, scope.EnvironmentName)
		query = injectLabelsIntoQuery(query, labels)

		backend.Logger.Debug("OTel scope injected into LogQL query",
			"original", args["query"],
			"injected", query,
			"scope", scope,
		)
	}

	// Note: Datasource type validation is skipped here because it requires an HTTP request
	// for authentication. If the datasource type doesn't match the query type,
	// Grafana's query API will return an appropriate error.

	// Build query payload
	queryPayload := QueryPayload{
		Query:         query,
		MaxDataPoints: int64(limit),
	}

	// Build time range
	timeRange := TimeRange{
		From: from,
		To:   to,
	}

	// Execute query via Grafana query client
	if a.grafanaQueryClient == nil {
		return "", fmt.Errorf("Grafana query client not initialized")
	}

	response, err := a.grafanaQueryClient.ExecuteQuery(ctx, user, datasourceUid, "loki", queryPayload, timeRange)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %w", err)
	}

	// Check for errors in response
	if result, ok := response.Results["A"]; ok && result.Error != "" {
		return "", fmt.Errorf("query error: %s", result.Error)
	}

	// Format response as structured JSON with log analysis
	result := formatLogQLResult(response, query)

	// Marshal to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// executeTraceQL executes a TraceQL query and returns trace analytics
func (a *App) executeTraceQL(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	// Extract parameters
	datasourceUid, ok := args["datasourceUid"].(string)
	if !ok || datasourceUid == "" {
		return "", fmt.Errorf("datasourceUid is required")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	from, ok := args["from"].(string)
	if !ok || from == "" {
		from = "now-15m"
	}

	to, ok := args["to"].(string)
	if !ok || to == "" {
		to = "now"
	}

	// Limit is optional - default to 100
	limit := 100
	if limitVal, ok := args["limit"].(float64); ok {
		limit = int(limitVal)
	}
	// Cap at 1000
	if limit > 1000 {
		limit = 1000
	}

	// Apply OTel enforcement if enabled
	if a.settings != nil && a.settings.OtelEnforcement.Enabled {
		// Extract service name and environment from args
		serviceName, _ := args["serviceName"].(string)
		environmentName, _ := args["environmentName"].(string)

		scope := &OtelScope{
			ServiceName:     serviceName,
			EnvironmentName: environmentName,
			Source:          "tool_args",
		}

		// Apply defaults
		a.applyOtelScopeDefaults(scope)

		// Validate scope
		if err := a.validateOtelScope(scope); err != nil {
			return "", fmt.Errorf("OTel validation failed: %w", err)
		}

		// Inject scope into query
		labelFormat := a.otelRegistry.GetFormat(datasourceUid)
		if labelFormat == nil {
			labelFormat = a.DiscoverOTelLabels(ctx, datasourceUid, DatasourceTempo)
			a.otelRegistry.SetFormat(datasourceUid, labelFormat)
		}

		selectors := BuildOTelLabels(labelFormat, scope.ServiceName, scope.EnvironmentName)
		query = injectTraceQLSelectors(query, selectors)

		backend.Logger.Debug("OTel scope injected into TraceQL query",
			"original", args["query"],
			"injected", query,
			"scope", scope,
		)
	}

	// Build query payload for Tempo
	queryPayload := QueryPayload{
		Query:         query,
		MaxDataPoints: int64(limit),
	}

	// Build time range
	timeRange := TimeRange{
		From: from,
		To:   to,
	}

	// Execute query via Grafana query client
	if a.grafanaQueryClient == nil {
		return "", fmt.Errorf("Grafana query client not initialized")
	}

	response, err := a.grafanaQueryClient.ExecuteQuery(ctx, user, datasourceUid, "tempo", queryPayload, timeRange)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %w", err)
	}

	// Check for errors in response
	if result, ok := response.Results["A"]; ok && result.Error != "" {
		return "", fmt.Errorf("query error: %s", result.Error)
	}

	// Format response as structured JSON with trace analytics
	result := formatTraceQLResult(response, query)

	// Marshal to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// PromQLResult represents structured PromQL query results with analytics
type PromQLResult struct {
	Query           string                 `json:"query"`
	DataPoints      int                    `json:"dataPoints"`
	Series          int                    `json:"series"`
	Min             float64                `json:"min"`
	Max             float64                `json:"max"`
	Avg             float64                `json:"avg"`
	Latest          float64                `json:"latest"`
	Trend           string                 `json:"trend"` // stable | increasing | decreasing | sharply_increasing | sharply_decreasing
	AnomalyDetected bool                   `json:"anomalyDetected"`
	TimeSeries      []TimeSeriesPoint      `json:"timeSeries,omitempty"`
	Labels          map[string]interface{} `json:"labels,omitempty"`
}

// TimeSeriesPoint represents a single data point
type TimeSeriesPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// LogQLResult represents structured LogQL query results with log analysis
type LogQLResult struct {
	Query         string                 `json:"query"`
	TotalLines    int                    `json:"totalLines"`
	ReturnedLines int                    `json:"returnedLines"`
	ErrorCount    int                    `json:"errorCount"`
	ErrorRate     float64                `json:"errorRate"`
	TopErrors     []ErrorPattern         `json:"topErrors"`
	TopLabels     map[string]interface{} `json:"topLabels"`
	FirstSeenAt   string                 `json:"firstSeenAt,omitempty"`
	Trend         string                 `json:"trend"` // stable | increasing | decreasing
	Anomaly       bool                   `json:"anomaly"`
}

// ErrorPattern represents a common error pattern
type ErrorPattern struct {
	Message string  `json:"message"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TraceQLResult represents structured TraceQL query results with trace analytics
type TraceQLResult struct {
	Query            string           `json:"query"`
	TraceCount       int              `json:"traceCount"`
	SpanCount        int              `json:"spanCount"`
	AvgDuration      float64          `json:"avgDuration"`      // milliseconds
	P50Duration      float64          `json:"p50Duration"`      // median
	P95Duration      float64          `json:"p95Duration"`
	P99Duration      float64          `json:"p99Duration"`
	MaxDuration      float64          `json:"maxDuration"`
	ErrorRate        float64          `json:"errorRate"`        // percentage
	ErrorCount       int              `json:"errorCount"`
	TopErrors        []ErrorPattern   `json:"topErrors"`
	TopServices      []ServiceStats   `json:"topServices"`
	TopOperations    []OperationStats `json:"topOperations"`
	SpanDistribution map[string]int   `json:"spanDistribution"` // duration buckets
	FirstSeenAt      string           `json:"firstSeenAt,omitempty"`
	Trend            string           `json:"trend"`   // stable | increasing | decreasing
	Anomaly          bool             `json:"anomaly"`
}

// ServiceStats represents service-level trace statistics
type ServiceStats struct {
	ServiceName string  `json:"serviceName"`
	SpanCount   int     `json:"spanCount"`
	AvgDuration float64 `json:"avgDuration"`
	ErrorRate   float64 `json:"errorRate"`
}

// OperationStats represents operation-level trace statistics
type OperationStats struct {
	OperationName string  `json:"operationName"`
	SpanCount     int     `json:"spanCount"`
	AvgDuration   float64 `json:"avgDuration"`
	P95Duration   float64 `json:"p95Duration"`
}

// formatPromQLResult formats Grafana query response into structured PromQL result
func formatPromQLResult(response *GrafanaQueryResponse, query string) *PromQLResult {
	result := &PromQLResult{
		Query:      query,
		DataPoints: 0,
		Series:     0,
		Min:        math.MaxFloat64,
		Max:        -math.MaxFloat64,
		Avg:        0,
		Latest:     0,
		Trend:      "stable",
	}

	resultData, ok := response.Results["A"]
	if !ok || len(resultData.Frames) == 0 {
		return result
	}

	var allValues []float64
	var latestValue float64
	var latestTime int64

	// Process frames
	for _, frame := range resultData.Frames {
		frameMap, ok := frame.(map[string]interface{})
		if !ok {
			continue
		}

		fields, ok := frameMap["schema"].(map[string]interface{})["fields"].([]interface{})
		if !ok {
			continue
		}

		// Find time and value fields
		var timeValues []int64
		var values []float64

		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			fieldName, _ := fieldMap["name"].(string)
			fieldValues, ok := fieldMap["values"].([]interface{})
			if !ok {
				continue
			}

			if fieldName == "Time" || fieldName == "time" {
				for _, v := range fieldValues {
					if t, ok := v.(float64); ok {
						timeValues = append(timeValues, int64(t))
					}
				}
			} else if fieldName == "Value" || fieldName == "value" {
				for _, v := range fieldValues {
					if val, ok := v.(float64); ok {
						values = append(values, val)
						allValues = append(allValues, val)
					}
				}
			}
		}

		// Find latest value
		for i, t := range timeValues {
			if i < len(values) && t > latestTime {
				latestTime = t
				latestValue = values[i]
			}
		}

		result.Series++
	}

	result.DataPoints = len(allValues)
	result.Latest = latestValue

	// Calculate statistics
	if len(allValues) > 0 {
		sum := 0.0
		for _, v := range allValues {
			if v < result.Min {
				result.Min = v
			}
			if v > result.Max {
				result.Max = v
			}
			sum += v
		}
		result.Avg = sum / float64(len(allValues))

		// Detect trend
		result.Trend = detectTrend(allValues)

		// Detect anomalies (>200% change)
		result.AnomalyDetected = detectAnomaly(allValues)

		// Downsample to max 20 points for TimeSeries
		downsampledValues := downsample(allValues, 20)
		result.TimeSeries = make([]TimeSeriesPoint, len(downsampledValues))
		for i, v := range downsampledValues {
			result.TimeSeries[i] = TimeSeriesPoint{
				Timestamp: time.Now().Add(-time.Duration(len(downsampledValues)-i) * time.Minute).Unix(),
				Value:     v,
			}
		}
	}

	return result
}

// formatLogQLResult formats Grafana query response into structured log analysis
func formatLogQLResult(response *GrafanaQueryResponse, query string) *LogQLResult {
	result := &LogQLResult{
		Query:         query,
		TotalLines:    0,
		ReturnedLines: 0,
		ErrorCount:    0,
		ErrorRate:     0,
		TopErrors:     []ErrorPattern{},
		TopLabels:     make(map[string]interface{}),
		Trend:         "stable",
		Anomaly:       false,
	}

	resultData, ok := response.Results["A"]
	if !ok || len(resultData.Frames) == 0 {
		return result
	}

	errorPatterns := make(map[string]int)
	labelCounts := make(map[string]map[string]int)

	// Process frames
	for _, frame := range resultData.Frames {
		frameMap, ok := frame.(map[string]interface{})
		if !ok {
			continue
		}

		fields, ok := frameMap["schema"].(map[string]interface{})["fields"].([]interface{})
		if !ok {
			continue
		}

		// Extract log lines and labels
		var logLines []string
		var timestamps []int64

		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			fieldName, _ := fieldMap["name"].(string)
			fieldValues, ok := fieldMap["values"].([]interface{})
			if !ok {
				continue
			}

			if fieldName == "Time" || fieldName == "time" {
				for _, v := range fieldValues {
					if t, ok := v.(float64); ok {
						timestamps = append(timestamps, int64(t))
					}
				}
			} else if fieldName == "Line" || fieldName == "line" {
				for _, v := range fieldValues {
					if line, ok := v.(string); ok {
						logLines = append(logLines, line)
					}
				}
			} else if fieldName == "labels" {
				// Extract label distribution
				for _, v := range fieldValues {
					if labelMap, ok := v.(map[string]interface{}); ok {
						for k, v := range labelMap {
							if labelCounts[k] == nil {
								labelCounts[k] = make(map[string]int)
							}
							if strVal, ok := v.(string); ok {
								labelCounts[k][strVal]++
							}
						}
					}
				}
			}
		}

		result.ReturnedLines += len(logLines)

		// Analyze log lines for errors
		for _, line := range logLines {
			lineLower := strings.ToLower(line)
			if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "exception") || strings.Contains(lineLower, "fatal") {
				result.ErrorCount++

				// Extract error pattern (first 100 chars)
				pattern := line
				if len(pattern) > 100 {
					pattern = pattern[:100]
				}
				errorPatterns[pattern]++
			}
		}

		// Set first seen timestamp
		if len(timestamps) > 0 && result.FirstSeenAt == "" {
			result.FirstSeenAt = time.Unix(timestamps[0]/1000, 0).UTC().Format(time.RFC3339)
		}
	}

	result.TotalLines = result.ReturnedLines

	// Calculate error rate
	if result.TotalLines > 0 {
		result.ErrorRate = float64(result.ErrorCount) / float64(result.TotalLines)
	}

	// Sort and get top 5 error patterns
	type errorCount struct {
		message string
		count   int
	}
	var errors []errorCount
	for msg, count := range errorPatterns {
		errors = append(errors, errorCount{msg, count})
	}
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})

	for i := 0; i < 5 && i < len(errors); i++ {
		percent := float64(errors[i].count) / float64(result.ErrorCount) * 100
		result.TopErrors = append(result.TopErrors, ErrorPattern{
			Message: errors[i].message,
			Count:   errors[i].count,
			Percent: percent,
		})
	}

	// Get top labels
	for labelName, valueCounts := range labelCounts {
		topValues := []string{}
		for value := range valueCounts {
			topValues = append(topValues, value)
		}
		if len(topValues) > 5 {
			topValues = topValues[:5]
		}
		result.TopLabels[labelName] = topValues
	}

	// Simple trend detection based on error rate
	if result.ErrorRate > 0.5 {
		result.Trend = "increasing"
		result.Anomaly = true
	} else if result.ErrorRate > 0.1 {
		result.Trend = "stable"
	} else {
		result.Trend = "decreasing"
	}

	return result
}

// formatTraceQLResult formats Grafana query response into structured trace analytics
func formatTraceQLResult(response *GrafanaQueryResponse, query string) *TraceQLResult {
	result := &TraceQLResult{
		Query:            query,
		TraceCount:       0,
		SpanCount:        0,
		AvgDuration:      0,
		P50Duration:      0,
		P95Duration:      0,
		P99Duration:      0,
		MaxDuration:      0,
		ErrorRate:        0,
		ErrorCount:       0,
		TopErrors:        []ErrorPattern{},
		TopServices:      []ServiceStats{},
		TopOperations:    []OperationStats{},
		SpanDistribution: make(map[string]int),
		Trend:            "stable",
		Anomaly:          false,
	}

	resultData, ok := response.Results["A"]
	if !ok || len(resultData.Frames) == 0 {
		return result
	}

	var allDurations []float64
	var firstTimestamp int64
	traceIDs := make(map[string]bool)
	serviceStats := make(map[string]*ServiceStats)
	operationStats := make(map[string]*OperationStats)
	errorMessages := make(map[string]int)

	// Process frames - Tempo returns span data in frames
	for _, frame := range resultData.Frames {
		frameMap, ok := frame.(map[string]interface{})
		if !ok {
			continue
		}

		schema, ok := frameMap["schema"].(map[string]interface{})
		if !ok {
			continue
		}

		fields, ok := schema["fields"].([]interface{})
		if !ok {
			continue
		}

		data, ok := frameMap["data"].(map[string]interface{})
		if !ok {
			continue
		}

		values, ok := data["values"].([]interface{})
		if !ok {
			continue
		}

		// Map field names to indices
		fieldIndices := make(map[string]int)
		for i, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := fieldMap["name"].(string)
			fieldIndices[name] = i
		}

		// Get values arrays
		var traceIDValues, spanIDValues, serviceValues, operationValues, durationValues, statusValues, startTimeValues []interface{}

		if idx, ok := fieldIndices["traceID"]; ok && idx < len(values) {
			traceIDValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["spanID"]; ok && idx < len(values) {
			spanIDValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["serviceName"]; ok && idx < len(values) {
			serviceValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["operationName"]; ok && idx < len(values) {
			operationValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["duration"]; ok && idx < len(values) {
			durationValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["statusCode"]; ok && idx < len(values) {
			statusValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["startTime"]; ok && idx < len(values) {
			startTimeValues, _ = values[idx].([]interface{})
		}

		// Process each span
		spanCount := 0
		if traceIDValues != nil {
			spanCount = len(traceIDValues)
		} else if spanIDValues != nil {
			spanCount = len(spanIDValues)
		} else if durationValues != nil {
			spanCount = len(durationValues)
		}

		for i := 0; i < spanCount; i++ {
			result.SpanCount++

			// Track unique trace IDs
			if traceIDValues != nil && i < len(traceIDValues) {
				if traceID, ok := traceIDValues[i].(string); ok && traceID != "" {
					traceIDs[traceID] = true
				}
			}

			// Duration - convert from nanoseconds to milliseconds
			var durationMs float64
			if durationValues != nil && i < len(durationValues) {
				if dur, ok := durationValues[i].(float64); ok {
					durationMs = dur / 1_000_000 // nanoseconds to milliseconds
					allDurations = append(allDurations, durationMs)

					if durationMs > result.MaxDuration {
						result.MaxDuration = durationMs
					}
				}
			}

			// Service stats
			var serviceName string
			if serviceValues != nil && i < len(serviceValues) {
				serviceName, _ = serviceValues[i].(string)
			}
			if serviceName != "" {
				if _, exists := serviceStats[serviceName]; !exists {
					serviceStats[serviceName] = &ServiceStats{ServiceName: serviceName}
				}
				serviceStats[serviceName].SpanCount++
				serviceStats[serviceName].AvgDuration += durationMs
			}

			// Operation stats
			var operationName string
			if operationValues != nil && i < len(operationValues) {
				operationName, _ = operationValues[i].(string)
			}
			if operationName != "" {
				if _, exists := operationStats[operationName]; !exists {
					operationStats[operationName] = &OperationStats{OperationName: operationName}
				}
				operationStats[operationName].SpanCount++
				operationStats[operationName].AvgDuration += durationMs
			}

			// Status code - 0 = OK, 1 = UNSET, 2 = ERROR
			if statusValues != nil && i < len(statusValues) {
				if status, ok := statusValues[i].(float64); ok && status == 2 {
					result.ErrorCount++
					// Use service + operation as error message pattern
					errorKey := serviceName + ": " + operationName
					if errorKey == ": " {
						errorKey = "Unknown error"
					}
					errorMessages[errorKey]++

					if serviceName != "" {
						serviceStats[serviceName].ErrorRate++
					}
				}
			}

			// First seen timestamp
			if startTimeValues != nil && i < len(startTimeValues) {
				if ts, ok := startTimeValues[i].(float64); ok {
					if firstTimestamp == 0 || int64(ts) < firstTimestamp {
						firstTimestamp = int64(ts)
					}
				}
			}

			// Duration distribution buckets
			if durationMs > 0 {
				bucket := getDurationBucket(durationMs)
				result.SpanDistribution[bucket]++
			}
		}
	}

	result.TraceCount = len(traceIDs)

	// Calculate duration percentiles
	if len(allDurations) > 0 {
		sort.Float64s(allDurations)

		// Average
		sum := 0.0
		for _, d := range allDurations {
			sum += d
		}
		result.AvgDuration = sum / float64(len(allDurations))

		// Percentiles
		result.P50Duration = percentile(allDurations, 50)
		result.P95Duration = percentile(allDurations, 95)
		result.P99Duration = percentile(allDurations, 99)

		// Trend detection
		result.Trend = detectTrend(allDurations)

		// Anomaly detection - any span >3x average
		for _, d := range allDurations {
			if d > result.AvgDuration*3 {
				result.Anomaly = true
				break
			}
		}
	}

	// Error rate
	if result.SpanCount > 0 {
		result.ErrorRate = float64(result.ErrorCount) / float64(result.SpanCount) * 100
	}

	// Finalize service stats (calculate average duration and error rate)
	for name, stats := range serviceStats {
		if stats.SpanCount > 0 {
			stats.AvgDuration = stats.AvgDuration / float64(stats.SpanCount)
			stats.ErrorRate = stats.ErrorRate / float64(stats.SpanCount) * 100
		}
		serviceStats[name] = stats
	}

	// Finalize operation stats
	operationDurations := make(map[string][]float64)
	for _, frame := range resultData.Frames {
		frameMap, ok := frame.(map[string]interface{})
		if !ok {
			continue
		}

		schema, ok := frameMap["schema"].(map[string]interface{})
		if !ok {
			continue
		}

		fields, ok := schema["fields"].([]interface{})
		if !ok {
			continue
		}

		data, ok := frameMap["data"].(map[string]interface{})
		if !ok {
			continue
		}

		values, ok := data["values"].([]interface{})
		if !ok {
			continue
		}

		fieldIndices := make(map[string]int)
		for i, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := fieldMap["name"].(string)
			fieldIndices[name] = i
		}

		var operationValues, durationValues []interface{}
		if idx, ok := fieldIndices["operationName"]; ok && idx < len(values) {
			operationValues, _ = values[idx].([]interface{})
		}
		if idx, ok := fieldIndices["duration"]; ok && idx < len(values) {
			durationValues, _ = values[idx].([]interface{})
		}

		if operationValues != nil && durationValues != nil {
			for i := 0; i < len(operationValues) && i < len(durationValues); i++ {
				opName, _ := operationValues[i].(string)
				dur, _ := durationValues[i].(float64)
				if opName != "" && dur > 0 {
					operationDurations[opName] = append(operationDurations[opName], dur/1_000_000)
				}
			}
		}
	}

	for name, stats := range operationStats {
		if stats.SpanCount > 0 {
			stats.AvgDuration = stats.AvgDuration / float64(stats.SpanCount)
		}
		if durations, ok := operationDurations[name]; ok && len(durations) > 0 {
			sort.Float64s(durations)
			stats.P95Duration = percentile(durations, 95)
		}
		operationStats[name] = stats
	}

	// Sort and get top 5 services by span count
	var services []ServiceStats
	for _, s := range serviceStats {
		services = append(services, *s)
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].SpanCount > services[j].SpanCount
	})
	if len(services) > 5 {
		services = services[:5]
	}
	result.TopServices = services

	// Sort and get top 5 operations by average duration (slowest)
	var operations []OperationStats
	for _, o := range operationStats {
		operations = append(operations, *o)
	}
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].AvgDuration > operations[j].AvgDuration
	})
	if len(operations) > 5 {
		operations = operations[:5]
	}
	result.TopOperations = operations

	// Sort and get top 5 errors
	type errorCount struct {
		message string
		count   int
	}
	var errors []errorCount
	for msg, count := range errorMessages {
		errors = append(errors, errorCount{msg, count})
	}
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})
	for i := 0; i < 5 && i < len(errors); i++ {
		pct := 0.0
		if result.ErrorCount > 0 {
			pct = float64(errors[i].count) / float64(result.ErrorCount) * 100
		}
		result.TopErrors = append(result.TopErrors, ErrorPattern{
			Message: errors[i].message,
			Count:   errors[i].count,
			Percent: pct,
		})
	}

	// First seen timestamp
	if firstTimestamp > 0 {
		result.FirstSeenAt = time.Unix(0, firstTimestamp).UTC().Format(time.RFC3339)
	}

	return result
}

// getDurationBucket returns the bucket name for a duration in milliseconds
func getDurationBucket(durationMs float64) string {
	switch {
	case durationMs < 10:
		return "<10ms"
	case durationMs < 50:
		return "10-50ms"
	case durationMs < 100:
		return "50-100ms"
	case durationMs < 500:
		return "100-500ms"
	default:
		return ">500ms"
	}
}

// percentile calculates the p-th percentile of a sorted slice
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)) * float64(p) / 100.0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// detectTrend analyzes values to determine trend
func detectTrend(values []float64) string {
	if len(values) < 3 {
		return "stable"
	}

	// Calculate average of first third vs last third
	thirdSize := len(values) / 3
	firstThird := values[:thirdSize]
	lastThird := values[len(values)-thirdSize:]

	firstAvg := average(firstThird)
	lastAvg := average(lastThird)

	if firstAvg == 0 {
		return "stable"
	}

	changePercent := ((lastAvg - firstAvg) / firstAvg) * 100

	if changePercent > 50 {
		return "sharply_increasing"
	} else if changePercent > 10 {
		return "increasing"
	} else if changePercent < -50 {
		return "sharply_decreasing"
	} else if changePercent < -10 {
		return "decreasing"
	}

	return "stable"
}

// detectAnomaly detects if any single step has >200% change
func detectAnomaly(values []float64) bool {
	for i := 1; i < len(values); i++ {
		if values[i-1] == 0 {
			continue
		}
		changePercent := math.Abs(((values[i] - values[i-1]) / values[i-1]) * 100)
		if changePercent > 200 {
			return true
		}
	}
	return false
}

// average calculates the average of a slice of float64
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// downsample reduces the number of data points to maxPoints
func downsample(values []float64, maxPoints int) []float64 {
	if len(values) <= maxPoints {
		return values
	}

	step := float64(len(values)) / float64(maxPoints)
	result := make([]float64, maxPoints)

	for i := 0; i < maxPoints; i++ {
		idx := int(float64(i) * step)
		if idx >= len(values) {
			idx = len(values) - 1
		}
		result[i] = values[idx]
	}

	return result
}

// parseDurationToMs converts duration strings like "15s", "1m", "5m" to milliseconds
func parseDurationToMs(duration string) (int64, error) {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return 0, err
	}
	return d.Milliseconds(), nil
}
