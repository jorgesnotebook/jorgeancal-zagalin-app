package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// AlertmanagerAlert represents a single alert from Grafana's Alertmanager API
type AlertmanagerAlert struct {
	Annotations map[string]string `json:"annotations"`
	EndsAt      string            `json:"endsAt"`
	Fingerprint string            `json:"fingerprint"`
	Receivers   []struct {
		Name string `json:"name"`
	} `json:"receivers"`
	StartsAt    string            `json:"startsAt"`
	Status      AlertStatus       `json:"status"`
	UpdatedAt   string            `json:"updatedAt"`
	GeneratorURL string           `json:"generatorURL"`
	Labels      map[string]string `json:"labels"`
}

// AlertStatus represents the status of an alert
type AlertStatus struct {
	InhibitedBy []string `json:"inhibitedBy"`
	SilencedBy  []string `json:"silencedBy"`
	State       string   `json:"state"`
}

// FiringAlertsResult represents the structured output for LLM consumption
type FiringAlertsResult struct {
	Count    int            `json:"count"`
	Alerts   []AlertSummary `json:"alerts"`
	Patterns []AlertPattern `json:"patterns,omitempty"`
}

// AlertSummary represents a simplified alert for LLM consumption
type AlertSummary struct {
	Name      string            `json:"name"`
	Severity  string            `json:"severity"`
	Service   string            `json:"service,omitempty"`
	Labels    map[string]string `json:"labels"`
	StartedAt string            `json:"startedAt"`
	Summary   string            `json:"summary,omitempty"`
}

// AlertPattern represents a detected pattern in alerts
type AlertPattern struct {
	Service    string `json:"service"`
	AlertCount int    `json:"alertCount"`
	Message    string `json:"message"`
}

// getFiringAlerts fetches currently firing alerts from Grafana's Alertmanager
func (a *App) getFiringAlerts(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	// Build URL for Alertmanager API
	grafanaURL := getGrafanaURL()
	url := fmt.Sprintf("%s/api/alertmanager/grafana/api/v2/alerts?active=true&silenced=false", grafanaURL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers using the same pattern as other Grafana calls
	if a.settings != nil && a.settings.ServiceAccountToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.settings.ServiceAccountToken))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Grafana-User", user.UserLogin)
	httpReq.Header.Set("X-Zagalin-User", user.UserLogin)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("alertmanager API failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var alerts []AlertmanagerAlert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	backend.Logger.Info("Fetched firing alerts",
		"count", len(alerts),
		"user", user.UserLogin,
	)

	// Format for LLM consumption
	result := formatFiringAlertsResult(alerts)

	// Marshal to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// formatFiringAlertsResult converts raw alerts to LLM-friendly format with pattern detection
func formatFiringAlertsResult(alerts []AlertmanagerAlert) *FiringAlertsResult {
	result := &FiringAlertsResult{
		Count:  len(alerts),
		Alerts: make([]AlertSummary, 0, len(alerts)),
	}

	// Track alerts per service for pattern detection
	serviceAlertCounts := make(map[string]int)

	for _, alert := range alerts {
		summary := AlertSummary{
			Name:      alert.Labels["alertname"],
			Severity:  getSeverity(alert.Labels),
			Service:   getService(alert.Labels),
			Labels:    filterRelevantLabels(alert.Labels),
			StartedAt: formatAlertTime(alert.StartsAt),
			Summary:   alert.Annotations["summary"],
		}

		result.Alerts = append(result.Alerts, summary)

		// Track service for pattern detection
		if summary.Service != "" {
			serviceAlertCounts[summary.Service]++
		}
	}

	// Detect patterns: services with multiple alerts
	result.Patterns = detectAlertPatterns(serviceAlertCounts)

	return result
}

// getSeverity extracts severity from alert labels with fallback
func getSeverity(labels map[string]string) string {
	// Common severity label names
	severityKeys := []string{"severity", "priority", "level"}
	for _, key := range severityKeys {
		if val, ok := labels[key]; ok && val != "" {
			return val
		}
	}
	return "unknown"
}

// getService extracts service name from alert labels
func getService(labels map[string]string) string {
	// Common service label names in order of preference
	serviceKeys := []string{
		"service",
		"service_name",
		"serviceName",
		"app",
		"application",
		"job",
		"namespace",
	}
	for _, key := range serviceKeys {
		if val, ok := labels[key]; ok && val != "" {
			return val
		}
	}
	return ""
}

// filterRelevantLabels removes internal/noisy labels and keeps relevant ones
func filterRelevantLabels(labels map[string]string) map[string]string {
	// Labels to exclude (internal Grafana/Alertmanager labels)
	excludeLabels := map[string]bool{
		"__alert_rule_uid__":         true,
		"__alert_rule_namespace_uid__": true,
		"grafana_folder":             true,
		"alertname":                  true, // Already extracted as Name
	}

	filtered := make(map[string]string)
	for k, v := range labels {
		if !excludeLabels[k] && v != "" {
			filtered[k] = v
		}
	}
	return filtered
}

// formatAlertTime formats the alert start time for readability
func formatAlertTime(startsAt string) string {
	t, err := time.Parse(time.RFC3339, startsAt)
	if err != nil {
		return startsAt
	}

	duration := time.Since(t)
	if duration < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

// detectAlertPatterns identifies services with multiple alerts
func detectAlertPatterns(serviceAlertCounts map[string]int) []AlertPattern {
	var patterns []AlertPattern

	// Sort services by alert count for consistent output
	type serviceCount struct {
		service string
		count   int
	}
	var sorted []serviceCount
	for service, count := range serviceAlertCounts {
		if count > 1 { // Only include services with multiple alerts
			sorted = append(sorted, serviceCount{service, count})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Create pattern messages
	for _, sc := range sorted {
		patterns = append(patterns, AlertPattern{
			Service:    sc.service,
			AlertCount: sc.count,
			Message:    fmt.Sprintf("%d alerts from %s — likely related", sc.count, sc.service),
		})
	}

	return patterns
}
