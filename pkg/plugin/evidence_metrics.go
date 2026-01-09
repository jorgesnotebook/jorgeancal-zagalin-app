package plugin

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

func SummarizeMetrics(
	frames []interface{},
	query string,
	datasource string,
	timeRange TimeRange,
) (*MetricsEvidence, error) {
	if len(frames) == 0 {
		return &MetricsEvidence{
			Quality: QualityNoData,
			Trend:   TrendNoData,
		}, nil
	}

	evidence := &MetricsEvidence{
		SeriesCount: len(frames),
		Quality:     QualityGood,
	}

	allValues := []float64{}
	timeSeriesData := [][]float64{}
	seriesLabels := []map[string]string{}

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

		var valueField map[string]interface{}
		var labels map[string]string

		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			fieldType, _ := fieldMap["type"].(string)
			if fieldType == "number" {
				valueField = fieldMap
				if labelMap, ok := fieldMap["labels"].(map[string]interface{}); ok {
					labels = make(map[string]string)
					for k, v := range labelMap {
						if strVal, ok := v.(string); ok {
							labels[k] = strVal
						}
					}
				}
			}
		}

		if valueField == nil {
			continue
		}

		data, ok := frameData["data"].(map[string]interface{})
		if !ok {
			continue
		}

		values, ok := data["values"].([]interface{})
		if !ok || len(values) == 0 {
			continue
		}

		valueColumn, ok := values[1].([]interface{})
		if !ok {
			continue
		}

		seriesValues := []float64{}
		for _, v := range valueColumn {
			if v == nil {
				continue
			}

			var val float64
			switch num := v.(type) {
			case float64:
				val = num
			case int:
				val = float64(num)
			case int64:
				val = float64(num)
			default:
				continue
			}

			if !math.IsNaN(val) && !math.IsInf(val, 0) {
				allValues = append(allValues, val)
				seriesValues = append(seriesValues, val)
			}
		}

		if len(seriesValues) > 0 {
			timeSeriesData = append(timeSeriesData, seriesValues)
			seriesLabels = append(seriesLabels, labels)
		}
	}

	if len(allValues) == 0 {
		return &MetricsEvidence{
			SeriesCount: evidence.SeriesCount,
			Quality:     QualityNoData,
			Trend:       TrendNoData,
		}, nil
	}

	evidence.Current = allValues[len(allValues)-1]
	evidence.Min = allValues[0]
	evidence.Max = allValues[0]
	sum := 0.0

	for _, val := range allValues {
		if val < evidence.Min {
			evidence.Min = val
		}
		if val > evidence.Max {
			evidence.Max = val
		}
		sum += val
	}

	evidence.Avg = sum / float64(len(allValues))

	evidence.Trend = detectTrend(allValues)
	evidence.SlopePerHour = calculateSlope(allValues, timeRange)

	if len(timeSeriesData) > 1 {
		evidence.TopContributors = extractTopContributors(timeSeriesData, seriesLabels, 5)
	}

	return evidence, nil
}

func detectTrend(values []float64) string {
	if len(values) < 2 {
		return TrendFlat
	}

	n := len(values)
	midpoint := n / 2

	firstHalf := values[:midpoint]
	secondHalf := values[midpoint:]

	firstAvg := average(firstHalf)
	secondAvg := average(secondHalf)

	variance := calculateVariance(values)
	mean := average(values)

	if variance > mean*0.5 {
		return TrendSpiky
	}

	diff := secondAvg - firstAvg
	threshold := mean * 0.1

	if diff > threshold {
		return TrendIncreasing
	} else if diff < -threshold {
		return TrendDecreasing
	}

	return TrendFlat
}

func calculateSlope(values []float64, timeRange TimeRange) float64 {
	if len(values) < 2 {
		return 0
	}

	n := float64(len(values))
	var sumX, sumY, sumXY, sumX2 float64

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	return slope * 3600
}

func extractTopContributors(
	seriesData [][]float64,
	labels []map[string]string,
	topN int,
) []LabelContributor {
	type seriesValue struct {
		labels map[string]string
		value  float64
	}

	contributors := []seriesValue{}

	for i, series := range seriesData {
		if len(series) == 0 {
			continue
		}

		currentValue := series[len(series)-1]
		contributors = append(contributors, seriesValue{
			labels: labels[i],
			value:  currentValue,
		})
	}

	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].value > contributors[j].value
	})

	if len(contributors) > topN {
		contributors = contributors[:topN]
	}

	result := make([]LabelContributor, len(contributors))
	for i, c := range contributors {
		result[i] = LabelContributor{
			Labels: c.labels,
			Value:  c.value,
		}
	}

	return result
}

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

func calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := average(values)
	sumSquares := 0.0

	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(values))
}

func BuildMetricsEvidencePack(
	grafanaResp *GrafanaQueryResponse,
	query string,
	datasource string,
	timeRange TimeRange,
) (*EvidencePack, error) {
	result, ok := grafanaResp.Results["A"]
	if !ok {
		return nil, fmt.Errorf("no results for refId A")
	}

	if result.Error != "" {
		return nil, fmt.Errorf("query error: %s", result.Error)
	}

	metrics, err := SummarizeMetrics(result.Frames, query, datasource, timeRange)
	if err != nil {
		return nil, err
	}

	return &EvidencePack{
		Type:       "metrics",
		Datasource: datasource,
		TimeRange:  timeRange,
		Query:      query,
		Metrics:    metrics,
		Quality:    metrics.Quality,
	}, nil
}
