package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func SummarizeLogs(
	frames []interface{},
	query string,
	datasource string,
	timeRange TimeRange,
) (*LogsEvidence, error) {
	if len(frames) == 0 {
		return &LogsEvidence{
			TotalCount:      0,
			Rate:            0,
			MaxRate:         0,
			Trend:           TrendNoData,
			TopLabels:       make(map[string][]string),
			NotableMessages: []string{},
		}, nil
	}

	evidence := &LogsEvidence{
		TopLabels:       make(map[string][]string),
		NotableMessages: []string{},
	}

	allMessages := []string{}
	labelCounts := make(map[string]map[string]int)
	timeStamps := []int64{}

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

		var timeFieldIdx, messageFieldIdx int = -1, -1
		var labels map[string]string

		for i, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			fieldName, _ := fieldMap["name"].(string)
			fieldType, _ := fieldMap["type"].(string)

			if fieldType == "time" {
				timeFieldIdx = i
			} else if fieldName == "line" || fieldName == "message" || fieldName == "log" {
				messageFieldIdx = i
			}

			if labelMap, ok := fieldMap["labels"].(map[string]interface{}); ok {
				labels = make(map[string]string)
				for k, v := range labelMap {
					if strVal, ok := v.(string); ok {
						labels[k] = strVal
					}
				}
			}
		}

		data, ok := frameData["data"].(map[string]interface{})
		if !ok {
			continue
		}

		values, ok := data["values"].([]interface{})
		if !ok || len(values) == 0 {
			continue
		}

		if timeFieldIdx >= 0 && timeFieldIdx < len(values) {
			if timeColumn, ok := values[timeFieldIdx].([]interface{}); ok {
				for _, t := range timeColumn {
					if ts, ok := t.(float64); ok {
						timeStamps = append(timeStamps, int64(ts))
					}
				}
			}
		}

		if messageFieldIdx >= 0 && messageFieldIdx < len(values) {
			if messageColumn, ok := values[messageFieldIdx].([]interface{}); ok {
				for _, msg := range messageColumn {
					if msgStr, ok := msg.(string); ok && msgStr != "" {
						allMessages = append(allMessages, msgStr)
					}
				}
			}
		}

		for labelKey, labelValue := range labels {
			if labelCounts[labelKey] == nil {
				labelCounts[labelKey] = make(map[string]int)
			}
			labelCounts[labelKey][labelValue]++
		}
	}

	evidence.TotalCount = int64(len(allMessages))

	if len(timeStamps) > 1 {
		sort.Slice(timeStamps, func(i, j int) bool {
			return timeStamps[i] < timeStamps[j]
		})

		startTime := timeStamps[0]
		endTime := timeStamps[len(timeStamps)-1]
		durationSeconds := float64(endTime-startTime) / 1000.0

		if durationSeconds > 0 {
			evidence.Rate = float64(evidence.TotalCount) / durationSeconds
		}

		evidence.MaxRate = calculateMaxLogRate(timeStamps)
		evidence.Trend = detectLogTrend(timeStamps)
	}

	evidence.TopLabels = extractTopLabels(labelCounts, 3)
	evidence.NotableMessages = extractNotableMessages(allMessages, 10)

	return evidence, nil
}

func calculateMaxLogRate(timestamps []int64) float64 {
	if len(timestamps) < 2 {
		return 0
	}

	bucketSize := int64(60000)
	buckets := make(map[int64]int)

	for _, ts := range timestamps {
		bucket := ts / bucketSize
		buckets[bucket]++
	}

	maxCount := 0
	for _, count := range buckets {
		if count > maxCount {
			maxCount = count
		}
	}

	return float64(maxCount) / 60.0
}

func detectLogTrend(timestamps []int64) string {
	if len(timestamps) < 10 {
		return TrendFlat
	}

	n := len(timestamps)
	midpoint := n / 2

	firstHalfRate := float64(midpoint) / (float64(timestamps[midpoint]-timestamps[0]) / 1000.0)
	secondHalfRate := float64(n-midpoint) / (float64(timestamps[n-1]-timestamps[midpoint]) / 1000.0)

	diff := secondHalfRate - firstHalfRate
	threshold := firstHalfRate * 0.2

	if diff > threshold {
		return TrendIncreasing
	} else if diff < -threshold {
		return TrendDecreasing
	}

	return TrendFlat
}

func extractTopLabels(labelCounts map[string]map[string]int, topN int) map[string][]string {
	result := make(map[string][]string)

	for labelKey, valueCounts := range labelCounts {
		type labelValue struct {
			value string
			count int
		}

		values := []labelValue{}
		for value, count := range valueCounts {
			values = append(values, labelValue{value: value, count: count})
		}

		sort.Slice(values, func(i, j int) bool {
			return values[i].count > values[j].count
		})

		topValues := []string{}
		for i := 0; i < len(values) && i < topN; i++ {
			topValues = append(topValues, values[i].value)
		}

		if len(topValues) > 0 {
			result[labelKey] = topValues
		}
	}

	return result
}

func extractNotableMessages(messages []string, maxMessages int) []string {
	if len(messages) == 0 {
		return []string{}
	}

	type messageScore struct {
		message string
		score   int
	}

	scores := []messageScore{}

	for _, msg := range messages {
		msgLower := strings.ToLower(msg)
		score := 0

		if strings.Contains(msgLower, "error") || strings.Contains(msgLower, "fatal") {
			score = 3
		} else if strings.Contains(msgLower, "warn") || strings.Contains(msgLower, "warning") {
			score = 2
		} else if strings.Contains(msgLower, "critical") || strings.Contains(msgLower, "exception") {
			score = 3
		} else {
			score = 1
		}

		scores = append(scores, messageScore{message: msg, score: score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	notable := []string{}
	seen := make(map[string]bool)

	for _, item := range scores {
		if len(notable) >= maxMessages {
			break
		}

		msgPreview := item.message
		if len(msgPreview) > 200 {
			msgPreview = msgPreview[:200] + "..."
		}

		signature := strings.ToLower(msgPreview[:min(50, len(msgPreview))])
		if !seen[signature] {
			notable = append(notable, msgPreview)
			seen[signature] = true
		}
	}

	return notable
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func BuildLogsEvidencePack(
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

	logs, err := SummarizeLogs(result.Frames, query, datasource, timeRange)
	if err != nil {
		return nil, err
	}

	quality := QualityGood
	if logs.TotalCount == 0 {
		quality = QualityNoData
	} else if logs.TotalCount < 10 {
		quality = QualityInsufficient
	}

	return &EvidencePack{
		Type:       "logs",
		Datasource: datasource,
		TimeRange:  timeRange,
		Query:      query,
		Logs:       logs,
		Quality:    quality,
	}, nil
}
