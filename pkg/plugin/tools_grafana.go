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

// DashboardSearchItem represents a single dashboard in search results
type DashboardSearchItem struct {
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	FolderTitle string   `json:"folderTitle,omitempty"`
	Tags        []string `json:"tags"`
	URL         string   `json:"url"`
}

// SearchDashboardsResult is the structured output for search_dashboards
type SearchDashboardsResult struct {
	Count      int                   `json:"count"`
	Dashboards []DashboardSearchItem `json:"dashboards"`
}

// DashboardPanelSummary is a simplified panel for LLM consumption
type DashboardPanelSummary struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// DashboardResult is the structured output for get_dashboard
type DashboardResult struct {
	UID         string                  `json:"uid"`
	Title       string                  `json:"title"`
	Tags        []string                `json:"tags"`
	FolderTitle string                  `json:"folderTitle,omitempty"`
	Panels      []DashboardPanelSummary `json:"panels"`
}

// AnnotationItem represents a single annotation
type AnnotationItem struct {
	Time int64    `json:"time"`
	Text string   `json:"text"`
	Tags []string `json:"tags"`
}

// AnnotationsResult is the structured output for get_annotations
type AnnotationsResult struct {
	Count       int              `json:"count"`
	Annotations []AnnotationItem `json:"annotations"`
}

// FolderItem represents a single Grafana folder
type FolderItem struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
}

// FoldersResult is the structured output for list_folders
type FoldersResult struct {
	Count   int          `json:"count"`
	Folders []FolderItem `json:"folders"`
}

// searchDashboards searches Grafana dashboards by query/tag
func (a *App) searchDashboards(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	grafanaURL := getGrafanaURL()

	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}
	tag := ""
	if t, ok := args["tag"].(string); ok {
		tag = t
	}
	limit := 50
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	url := fmt.Sprintf("%s/api/search?type=dash-db&limit=%d", grafanaURL, limit)
	if query != "" {
		url += "&query=" + query
	}
	if tag != "" {
		url += "&tag=" + tag
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if a.settings != nil && a.settings.ServiceAccountToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.settings.ServiceAccountToken))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Grafana-User", user.UserLogin)
	httpReq.Header.Set("X-Zagalin-User", user.UserLogin)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("search API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var raw []struct {
		UID         string   `json:"uid"`
		Title       string   `json:"title"`
		FolderTitle string   `json:"folderTitle"`
		Tags        []string `json:"tags"`
		URL         string   `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	result := &SearchDashboardsResult{
		Count:      len(raw),
		Dashboards: make([]DashboardSearchItem, 0, len(raw)),
	}
	for _, d := range raw {
		tags := d.Tags
		if tags == nil {
			tags = []string{}
		}
		result.Dashboards = append(result.Dashboards, DashboardSearchItem{
			UID:         d.UID,
			Title:       d.Title,
			FolderTitle: d.FolderTitle,
			Tags:        tags,
			URL:         d.URL,
		})
	}

	backend.Logger.Info("Searched dashboards", "count", result.Count, "user", user.UserLogin)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

// getDashboard fetches a specific dashboard by UID and returns a simplified summary
func (a *App) getDashboard(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	uid, ok := args["uid"].(string)
	if !ok || uid == "" {
		return "", fmt.Errorf("missing required parameter: uid")
	}

	grafanaURL := getGrafanaURL()
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", grafanaURL, uid)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if a.settings != nil && a.settings.ServiceAccountToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.settings.ServiceAccountToken))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Grafana-User", user.UserLogin)
	httpReq.Header.Set("X-Zagalin-User", user.UserLogin)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("dashboard API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Dashboard struct {
			UID    string   `json:"uid"`
			Title  string   `json:"title"`
			Tags   []string `json:"tags"`
			Panels []struct {
				ID    int    `json:"id"`
				Title string `json:"title"`
				Type  string `json:"type"`
			} `json:"panels"`
		} `json:"dashboard"`
		Meta struct {
			FolderTitle string `json:"folderTitle"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	tags := raw.Dashboard.Tags
	if tags == nil {
		tags = []string{}
	}

	panels := make([]DashboardPanelSummary, 0, len(raw.Dashboard.Panels))
	for _, p := range raw.Dashboard.Panels {
		panels = append(panels, DashboardPanelSummary{
			ID:    p.ID,
			Title: p.Title,
			Type:  p.Type,
		})
	}

	result := &DashboardResult{
		UID:         raw.Dashboard.UID,
		Title:       raw.Dashboard.Title,
		Tags:        tags,
		FolderTitle: raw.Meta.FolderTitle,
		Panels:      panels,
	}

	backend.Logger.Info("Fetched dashboard", "uid", uid, "user", user.UserLogin)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

// getAnnotations fetches annotations from Grafana
func (a *App) getAnnotations(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	grafanaURL := getGrafanaURL()

	from := "now-1h"
	if f, ok := args["from"].(string); ok && f != "" {
		from = f
	}
	to := "now"
	if t, ok := args["to"].(string); ok && t != "" {
		to = t
	}

	url := fmt.Sprintf("%s/api/annotations?from=%s&to=%s&limit=500", grafanaURL, from, to)
	if dashUID, ok := args["dashboardUID"].(string); ok && dashUID != "" {
		url += "&dashboardUID=" + dashUID
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if a.settings != nil && a.settings.ServiceAccountToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.settings.ServiceAccountToken))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Grafana-User", user.UserLogin)
	httpReq.Header.Set("X-Zagalin-User", user.UserLogin)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("annotations API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var raw []struct {
		Time int64    `json:"time"`
		Text string   `json:"text"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	result := &AnnotationsResult{
		Count:       len(raw),
		Annotations: make([]AnnotationItem, 0, len(raw)),
	}
	for _, a := range raw {
		tags := a.Tags
		if tags == nil {
			tags = []string{}
		}
		result.Annotations = append(result.Annotations, AnnotationItem{
			Time: a.Time,
			Text: a.Text,
			Tags: tags,
		})
	}

	backend.Logger.Info("Fetched annotations", "count", result.Count, "user", user.UserLogin)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

// listFolders lists all Grafana folders
func (a *App) listFolders(ctx context.Context, args map[string]interface{}, user *UserIdentity) (string, error) {
	grafanaURL := getGrafanaURL()
	url := fmt.Sprintf("%s/api/folders", grafanaURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if a.settings != nil && a.settings.ServiceAccountToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.settings.ServiceAccountToken))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Grafana-User", user.UserLogin)
	httpReq.Header.Set("X-Zagalin-User", user.UserLogin)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("folders API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var raw []struct {
		UID   string `json:"uid"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	result := &FoldersResult{
		Count:   len(raw),
		Folders: make([]FolderItem, 0, len(raw)),
	}
	for _, f := range raw {
		result.Folders = append(result.Folders, FolderItem{
			UID:   f.UID,
			Title: f.Title,
		})
	}

	backend.Logger.Info("Listed folders", "count", result.Count, "user", user.UserLogin)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
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
