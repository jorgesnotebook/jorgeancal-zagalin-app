package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// QueryValidationResult represents the outcome of validation
type QueryValidationResult struct {
	Valid          bool
	Sanitized      bool
	SanitizedQuery string
	Error          error
	ViolationType  string // "syntax", "injection", "complexity", "semantic", etc.
	OriginalQuery  string

	// LLM validation results
	LLMWarnings    []string // Advisory warnings from LLM
	LLMSuggestions []string // Improvement suggestions from LLM
	LLMBlocked     bool     // True if LLM blocked the query in strict mode
}

// QueryValidator handles query validation for all datasource types
type QueryValidator struct {
	settings *QueryValidationSettings
	app      *App // Reference to app for LLM access
}

// NewQueryValidator creates a new query validator
func NewQueryValidator(settings *QueryValidationSettings, app *App) *QueryValidator {
	if settings == nil {
		settings = &QueryValidationSettings{
			Enabled:               false,
			StrictMode:            false,
			MaxQueryComplexity:    100,
			LogValidationAttempts: true,
			EnableLLMValidation:   false,
			LLMValidationMode:     "advisory",
		}
	}
	return &QueryValidator{
		settings: settings,
		app:      app,
	}
}

// ValidateQuery validates a query based on datasource type
// This performs both parser-based validation (syntax, injection) and LLM-based validation (semantic)
func (v *QueryValidator) ValidateQuery(ctx context.Context, query string, dsType DatasourceType) *QueryValidationResult {
	if !v.settings.Enabled {
		return &QueryValidationResult{Valid: true}
	}

	// Phase 1: Parser-based validation (security-critical)
	var parserResult *QueryValidationResult
	switch dsType {
	case DatasourcePrometheus:
		// Check if PromQL validation is enabled
		if !v.settings.EnablePromQLValidation {
			return &QueryValidationResult{Valid: true}
		}
		parserResult = v.validatePromQL(query)
	case DatasourceLoki:
		// Check if LogQL validation is enabled
		if !v.settings.EnableLogQLValidation {
			return &QueryValidationResult{Valid: true}
		}
		parserResult = v.validateLogQL(query)
	case DatasourceTempo:
		// Check if TraceQL validation is enabled
		if !v.settings.EnableTraceQLValidation {
			return &QueryValidationResult{Valid: true}
		}
		parserResult = v.validateTraceQL(query)
	default:
		// For unknown datasources, perform basic string validation only
		parserResult = v.validateGeneric(query)
	}

	// If parser validation failed, return immediately (security layer)
	if !parserResult.Valid {
		return parserResult
	}

	// Phase 2: LLM semantic validation (optional, advisory or strict)
	if v.settings.EnableLLMValidation {
		llmResult := v.validateWithLLM(ctx, query, dsType, parserResult.SanitizedQuery)

		// Merge LLM results into parser result
		parserResult.LLMWarnings = llmResult.LLMWarnings
		parserResult.LLMSuggestions = llmResult.LLMSuggestions
		parserResult.LLMBlocked = llmResult.LLMBlocked

		// If LLM validation is in strict mode and blocked the query, mark as invalid
		if v.settings.LLMValidationMode == "strict" && llmResult.LLMBlocked {
			parserResult.Valid = false
			parserResult.Error = fmt.Errorf("query blocked by semantic validation: %s", strings.Join(llmResult.LLMWarnings, "; "))
			parserResult.ViolationType = "semantic"
		}
	}

	return parserResult
}

// validatePromQL validates PromQL syntax using manual pattern checking
// Note: This implementation uses manual validation instead of the Prometheus parser
// to avoid external dependencies. It validates common PromQL patterns and syntax.
func (v *QueryValidator) validatePromQL(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true, // Start optimistic
	}

	// First, apply generic validation (length, dangerous patterns)
	genericResult := v.validateGeneric(query)
	if !genericResult.Valid {
		return genericResult
	}

	// PromQL-specific syntax validation
	query = strings.TrimSpace(query)

	if len(query) == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("empty PromQL query")
		result.ViolationType = "syntax"
		return result
	}

	// Check balanced braces/brackets/parentheses
	if !v.hasBalancedBraces(query) {
		if v.settings.StrictMode {
			result.Valid = false
			result.Error = fmt.Errorf("unbalanced braces in PromQL query")
			result.ViolationType = "syntax"
			return result
		}

		// Attempt sanitization
		sanitized := v.sanitizePromQL(query)
		if !v.hasBalancedBraces(sanitized) {
			result.Valid = false
			result.Error = fmt.Errorf("invalid PromQL syntax: unbalanced braces")
			result.ViolationType = "syntax"
			return result
		}

		result.Sanitized = true
		result.SanitizedQuery = sanitized
		backend.Logger.Info("PromQL sanitized successfully",
			"originalHash", hashQuery(query),
			"sanitizedHash", hashQuery(sanitized),
			"originalLength", len(query),
			"sanitizedLength", len(sanitized))
		return result
	}

	// Check for invalid PromQL operators
	invalidOperatorPatterns := []string{
		"===",  // JavaScript-style
		"!==",  // JavaScript-style
		"<>",   // SQL-style not equal
		"++",   // C-style increment
		"--",   // C-style decrement (but -- is used in comments, so be careful)
	}

	for _, pattern := range invalidOperatorPatterns {
		if strings.Contains(query, pattern) {
			if v.settings.StrictMode {
				result.Valid = false
				result.Error = fmt.Errorf("invalid operator '%s' in PromQL query", pattern)
				result.ViolationType = "syntax"
				return result
			}
		}
	}

	// Estimate complexity
	complexity := v.countPromQLComplexityManual(query)
	if complexity > v.settings.MaxQueryComplexity {
		result.Valid = false
		result.Error = fmt.Errorf("query too complex: estimated %d nodes (max: %d)", complexity, v.settings.MaxQueryComplexity)
		result.ViolationType = "complexity"
		return result
	}

	// Check function allowlist if configured
	if len(v.settings.AllowedFunctions) > 0 {
		if err := v.checkPromQLFunctionsManual(query); err != nil {
			result.Valid = false
			result.Error = err
			result.ViolationType = "function_blocked"
			return result
		}
	}

	backend.Logger.Debug("PromQL validation passed",
		"query", query,
		"complexity", complexity)

	result.Valid = true
	return result
}

// validateLogQL validates LogQL syntax using manual pattern checking
// Note: This implementation uses manual validation instead of the Loki parser
// to avoid external dependencies. It validates common LogQL patterns and syntax.
func (v *QueryValidator) validateLogQL(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true, // Start optimistic
	}

	// First, apply generic validation (length, dangerous patterns)
	genericResult := v.validateGeneric(query)
	if !genericResult.Valid {
		return genericResult
	}

	// LogQL-specific syntax validation
	query = strings.TrimSpace(query)

	if len(query) == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("empty LogQL query")
		result.ViolationType = "syntax"
		return result
	}

	// Check balanced braces
	if !v.hasBalancedBraces(query) {
		if v.settings.StrictMode {
			result.Valid = false
			result.Error = fmt.Errorf("unbalanced braces in LogQL query")
			result.ViolationType = "syntax"
			return result
		}

		// Attempt sanitization
		sanitized := v.sanitizeLogQL(query)
		if !v.hasBalancedBraces(sanitized) {
			result.Valid = false
			result.Error = fmt.Errorf("invalid LogQL syntax: unbalanced braces")
			result.ViolationType = "syntax"
			return result
		}

		result.Sanitized = true
		result.SanitizedQuery = sanitized
		backend.Logger.Info("LogQL sanitized successfully",
			"originalHash", hashQuery(query),
			"sanitizedHash", hashQuery(sanitized),
			"originalLength", len(query),
			"sanitizedLength", len(sanitized))
		return result
	}

	// LogQL queries must start with a log selector { }
	// Valid patterns: {job="foo"}, {job="foo"} |= "bar", {job="foo"} | json, etc.
	hasLogSelector := strings.Contains(query, "{") && strings.Contains(query, "}")

	if !hasLogSelector && v.settings.StrictMode {
		result.Valid = false
		result.Error = fmt.Errorf("LogQL query must contain a log stream selector {}")
		result.ViolationType = "syntax"
		return result
	}

	// Check for invalid operators
	invalidOperatorPatterns := []string{
		"===",  // JavaScript-style
		"!==",  // JavaScript-style
		"<>",   // SQL-style not equal
	}

	for _, pattern := range invalidOperatorPatterns {
		if strings.Contains(query, pattern) {
			if v.settings.StrictMode {
				result.Valid = false
				result.Error = fmt.Errorf("invalid operator '%s' in LogQL query", pattern)
				result.ViolationType = "syntax"
				return result
			}
		}
	}

	// Estimate complexity
	complexity := v.countLogQLComplexityManual(query)
	if complexity > v.settings.MaxQueryComplexity {
		result.Valid = false
		result.Error = fmt.Errorf("query too complex: estimated %d nodes (max: %d)", complexity, v.settings.MaxQueryComplexity)
		result.ViolationType = "complexity"
		return result
	}

	backend.Logger.Debug("LogQL validation passed",
		"query", query,
		"complexity", complexity)

	result.Valid = true
	return result
}

// validateTraceQL validates TraceQL syntax using manual pattern checking
// Note: This implementation uses manual validation instead of the Tempo parser
// to avoid dependency conflicts. It validates common TraceQL patterns and syntax.
func (v *QueryValidator) validateTraceQL(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true, // Start optimistic
	}

	// First, apply generic validation (length, dangerous patterns)
	genericResult := v.validateGeneric(query)
	if !genericResult.Valid {
		return genericResult
	}

	// TraceQL-specific syntax validation
	query = strings.TrimSpace(query)

	// Check for basic TraceQL structure
	// TraceQL queries typically have format: { selector } or { selector } | aggregations
	if len(query) == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("empty TraceQL query")
		result.ViolationType = "syntax"
		return result
	}

	// Check balanced braces
	if !v.hasBalancedBraces(query) {
		if v.settings.StrictMode {
			result.Valid = false
			result.Error = fmt.Errorf("unbalanced braces in TraceQL query")
			result.ViolationType = "syntax"
			return result
		}

		// Attempt sanitization
		sanitized := v.sanitizeTraceQL(query)
		if !v.hasBalancedBraces(sanitized) {
			result.Valid = false
			result.Error = fmt.Errorf("invalid TraceQL syntax: unbalanced braces")
			result.ViolationType = "syntax"
			return result
		}

		result.Sanitized = true
		result.SanitizedQuery = sanitized
		backend.Logger.Info("TraceQL sanitized successfully",
			"originalHash", hashQuery(query),
			"sanitizedHash", hashQuery(sanitized),
			"originalLength", len(query),
			"sanitizedLength", len(sanitized))
		return result
	}

	// Check for valid TraceQL attribute prefixes
	// Common: span., resource., .field, intrinsic fields (name, duration, status, etc.)
	validPrefixes := []string{"span.", "resource.", "name", "duration", "status", "kind", "rootName", "rootServiceName"}
	hasValidPrefix := false

	// Simple heuristic: if query contains { }, check for valid attribute references
	if strings.Contains(query, "{") && strings.Contains(query, "}") {
		for _, prefix := range validPrefixes {
			if strings.Contains(query, prefix) {
				hasValidPrefix = true
				break
			}
		}

		// Also check for bare field references like { .http.status_code = 200 }
		if strings.Contains(query, ".") {
			hasValidPrefix = true
		}

		// Empty selector {} is valid in TraceQL
		if strings.Contains(query, "{}") {
			hasValidPrefix = true
		}
	} else {
		// No selectors, might be intrinsic field query
		hasValidPrefix = true
	}

	if !hasValidPrefix && v.settings.StrictMode {
		result.Valid = false
		result.Error = fmt.Errorf("TraceQL query must contain valid attribute selectors (span., resource., or intrinsic fields)")
		result.ViolationType = "syntax"
		return result
	}

	// Check for valid TraceQL operators
	// Valid: = != > < >= <= =~ !~
	invalidOperatorPatterns := []string{
		"===",  // JavaScript-style
		"!==",  // JavaScript-style
		"<>",   // SQL-style not equal
	}

	for _, pattern := range invalidOperatorPatterns {
		if strings.Contains(query, pattern) {
			if v.settings.StrictMode {
				result.Valid = false
				result.Error = fmt.Errorf("invalid operator '%s' in TraceQL query", pattern)
				result.ViolationType = "syntax"
				return result
			}
		}
	}

	// Estimate complexity based on query structure
	complexity := v.countTraceQLComplexity(query)
	if complexity > v.settings.MaxQueryComplexity {
		result.Valid = false
		result.Error = fmt.Errorf("query too complex: estimated %d nodes (max: %d)", complexity, v.settings.MaxQueryComplexity)
		result.ViolationType = "complexity"
		return result
	}

	backend.Logger.Debug("TraceQL validation passed",
		"query", query,
		"complexity", complexity)

	return result
}

// hasBalancedBraces checks if braces, brackets, and parentheses are balanced
func (v *QueryValidator) hasBalancedBraces(query string) bool {
	counts := map[rune]int{
		'{': 0,
		'[': 0,
		'(': 0,
	}

	inString := false
	var stringDelim rune

	for i, ch := range query {
		// Handle string literals
		if ch == '"' || ch == '\'' || ch == '`' {
			if !inString {
				inString = true
				stringDelim = ch
			} else if ch == stringDelim {
				// Check if escaped
				if i > 0 && query[i-1] != '\\' {
					inString = false
				}
			}
			continue
		}

		if inString {
			continue
		}

		// Count opening braces
		if ch == '{' || ch == '[' || ch == '(' {
			counts[ch]++
		}

		// Count closing braces
		if ch == '}' {
			counts['{']--
		} else if ch == ']' {
			counts['[']--
		} else if ch == ')' {
			counts['(']--
		}
	}

	// All counts should be zero
	return counts['{'] == 0 && counts['['] == 0 && counts['('] == 0
}

// sanitizeTraceQL attempts to fix common TraceQL issues
func (v *QueryValidator) sanitizeTraceQL(query string) string {
	// Remove leading/trailing whitespace
	sanitized := strings.TrimSpace(query)

	// Additional sanitization could be added here
	// For now, just trimming is safest

	return sanitized
}

// countTraceQLComplexity estimates query complexity based on query structure
func (v *QueryValidator) countTraceQLComplexity(query string) int {
	complexity := 1 // Base complexity

	// Count selectors (braces)
	complexity += strings.Count(query, "{")

	// Count logical operators
	complexity += strings.Count(query, "&&")
	complexity += strings.Count(query, "||")

	// Count comparison operators
	complexity += strings.Count(query, "=")
	complexity += strings.Count(query, "!=")
	complexity += strings.Count(query, ">")
	complexity += strings.Count(query, "<")
	complexity += strings.Count(query, "=~")
	complexity += strings.Count(query, "!~")

	// Count aggregations/pipelines
	complexity += strings.Count(query, "|")

	// Count function-like patterns (rough heuristic)
	complexity += strings.Count(query, "(")

	return complexity
}

// validateGeneric performs basic validation for unknown datasources
func (v *QueryValidator) validateGeneric(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true,
	}

	// Basic checks:
	// 1. Query length
	if len(query) > 10000 {
		result.Valid = false
		result.Error = fmt.Errorf("query too long: %d chars (max: 10000)", len(query))
		result.ViolationType = "length"
		return result
	}

	// 2. Dangerous patterns (SQL injection-like patterns)
	dangerousPatterns := []string{
		"; DROP ",
		"; DELETE ",
		"'; --",
		"/**/",
		"UNION SELECT",
		"<script",
		"javascript:",
	}

	upperQuery := strings.ToUpper(query)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(upperQuery, pattern) {
			result.Valid = false
			result.Error = fmt.Errorf("query contains potentially dangerous pattern")
			result.ViolationType = "injection"
			return result
		}
	}

	return result
}

// sanitizePromQL attempts to fix common PromQL issues
func (v *QueryValidator) sanitizePromQL(query string) string {
	// Remove leading/trailing whitespace
	sanitized := strings.TrimSpace(query)

	// Additional sanitization could be added here
	// For now, just trimming is safest

	return sanitized
}

// sanitizeLogQL attempts to fix common LogQL issues
func (v *QueryValidator) sanitizeLogQL(query string) string {
	// Remove leading/trailing whitespace
	sanitized := strings.TrimSpace(query)

	// Additional sanitization could be added here
	// For now, just trimming is safest

	return sanitized
}

// countPromQLComplexityManual estimates PromQL complexity based on query structure
func (v *QueryValidator) countPromQLComplexityManual(query string) int {
	complexity := 1 // Base complexity

	// Count metric selectors (braces)
	complexity += strings.Count(query, "{")

	// Count logical operators
	complexity += strings.Count(query, "and")
	complexity += strings.Count(query, "or")
	complexity += strings.Count(query, "unless")

	// Count comparison operators
	complexity += strings.Count(query, "==")
	complexity += strings.Count(query, "!=")
	complexity += strings.Count(query, ">")
	complexity += strings.Count(query, "<")
	complexity += strings.Count(query, "=~")
	complexity += strings.Count(query, "!~")

	// Count arithmetic operators
	complexity += strings.Count(query, "+")
	complexity += strings.Count(query, "-")
	complexity += strings.Count(query, "*")
	complexity += strings.Count(query, "/")
	complexity += strings.Count(query, "%")
	complexity += strings.Count(query, "^")

	// Count aggregations (common PromQL functions)
	complexity += strings.Count(query, "sum")
	complexity += strings.Count(query, "avg")
	complexity += strings.Count(query, "min")
	complexity += strings.Count(query, "max")
	complexity += strings.Count(query, "count")

	// Count function calls (parentheses)
	complexity += strings.Count(query, "(")

	// Count range vectors (square brackets)
	complexity += strings.Count(query, "[")

	return complexity
}

// countLogQLComplexityManual estimates LogQL complexity based on query structure
func (v *QueryValidator) countLogQLComplexityManual(query string) int {
	complexity := 1 // Base complexity

	// Count log selectors (braces)
	complexity += strings.Count(query, "{")

	// Count filter operators
	complexity += strings.Count(query, "|=")
	complexity += strings.Count(query, "!=")
	complexity += strings.Count(query, "|~")
	complexity += strings.Count(query, "!~")

	// Count parser operators
	complexity += strings.Count(query, "| json")
	complexity += strings.Count(query, "| logfmt")
	complexity += strings.Count(query, "| pattern")
	complexity += strings.Count(query, "| regexp")

	// Count aggregations
	complexity += strings.Count(query, "sum")
	complexity += strings.Count(query, "avg")
	complexity += strings.Count(query, "min")
	complexity += strings.Count(query, "max")
	complexity += strings.Count(query, "count")
	complexity += strings.Count(query, "rate")

	// Count function calls
	complexity += strings.Count(query, "(")

	// Count range operations
	complexity += strings.Count(query, "[")

	return complexity
}

// checkPromQLFunctionsManual validates function usage against allowlist using pattern matching
func (v *QueryValidator) checkPromQLFunctionsManual(query string) error {
	if len(v.settings.AllowedFunctions) == 0 {
		return nil
	}

	// Common PromQL functions to check
	commonFunctions := []string{
		"rate", "irate", "increase", "delta", "idelta",
		"sum", "avg", "min", "max", "count", "stddev", "stdvar",
		"topk", "bottomk", "quantile",
		"histogram_quantile", "histogram_count", "histogram_sum",
		"abs", "ceil", "floor", "round", "sqrt", "exp", "ln", "log2", "log10",
		"sort", "sort_desc",
		"time", "timestamp",
		"label_replace", "label_join",
		"vector", "scalar",
		"avg_over_time", "min_over_time", "max_over_time", "sum_over_time", "count_over_time",
		"quantile_over_time", "stddev_over_time", "stdvar_over_time",
	}

	var violations []string
	violationSet := make(map[string]bool) // Track unique violations

	for _, funcName := range commonFunctions {
		// Check if function is used in query (simple pattern matching)
		// Look for function( pattern with optional whitespace
		pattern := funcName + "("
		if strings.Contains(query, pattern) {
			// Check if it's in allowlist
			allowed := false
			for _, allowedFunc := range v.settings.AllowedFunctions {
				if allowedFunc == funcName {
					allowed = true
					break
				}
			}
			if !allowed && !violationSet[funcName] {
				violations = append(violations, funcName)
				violationSet[funcName] = true
			}
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("disallowed functions used: %s", strings.Join(violations, ", "))
	}

	return nil
}

// validateWithLLM performs semantic validation using the LLM
// This checks for expensive queries, best practices, and suggests improvements
func (v *QueryValidator) validateWithLLM(ctx context.Context, originalQuery string, dsType DatasourceType, sanitizedQuery string) *QueryValidationResult {
	result := &QueryValidationResult{
		Valid:         true,
		OriginalQuery: originalQuery,
	}

	// Use the sanitized query if available, otherwise original
	queryToValidate := originalQuery
	if sanitizedQuery != "" {
		queryToValidate = sanitizedQuery
	}

	// Build LLM prompt for semantic validation
	prompt := v.buildLLMValidationPrompt(queryToValidate, dsType)

	// Call LLM (this is a simplified version - real implementation would use the app's LLM service)
	llmResponse, err := v.callLLMForValidation(ctx, prompt)
	if err != nil {
		backend.Logger.Warn("LLM validation failed", "error", err)
		// Don't fail the query if LLM is unavailable - this is advisory only
		return result
	}

	// Parse LLM response
	v.parseLLMValidationResponse(llmResponse, result)

	return result
}

// buildLLMValidationPrompt creates a prompt for LLM semantic validation
func (v *QueryValidator) buildLLMValidationPrompt(query string, dsType DatasourceType) string {
	var queryLanguage string
	switch dsType {
	case DatasourcePrometheus:
		queryLanguage = "PromQL"
	case DatasourceLoki:
		queryLanguage = "LogQL"
	case DatasourceTempo:
		queryLanguage = "TraceQL"
	default:
		queryLanguage = "query"
	}

	return fmt.Sprintf(`You are a query validation expert. Analyze this %s query for semantic issues:

Query: %s

Please analyze for:
1. Query performance concerns (expensive operations, large time ranges, high cardinality)
2. Best practice violations
3. Potential improvements
4. Security concerns (beyond syntax validation)

Respond in JSON format:
{
  "safe": true/false,
  "warnings": ["warning1", "warning2"],
  "suggestions": ["suggestion1", "suggestion2"],
  "reason": "explanation if not safe"
}`, queryLanguage, query)
}

// callLLMForValidation calls the LLM service for semantic validation
func (v *QueryValidator) callLLMForValidation(ctx context.Context, prompt string) (string, error) {
	// TODO: Integrate with actual LLM service through grafana-llm-app
	// For now, return a placeholder
	// This would call the LLM API similar to how the chat functionality works

	backend.Logger.Debug("LLM validation would be called here", "prompt", prompt)

	// Placeholder response
	return `{"safe": true, "warnings": [], "suggestions": [], "reason": ""}`, nil
}

// parseLLMValidationResponse parses the LLM's validation response
func (v *QueryValidator) parseLLMValidationResponse(response string, result *QueryValidationResult) {
	var llmResp struct {
		Safe        bool     `json:"safe"`
		Warnings    []string `json:"warnings"`
		Suggestions []string `json:"suggestions"`
		Reason      string   `json:"reason"`
	}

	err := json.Unmarshal([]byte(response), &llmResp)
	if err != nil {
		backend.Logger.Warn("Failed to parse LLM validation response", "error", err)
		return
	}

	result.LLMWarnings = llmResp.Warnings
	result.LLMSuggestions = llmResp.Suggestions

	if !llmResp.Safe {
		result.LLMBlocked = true
		if llmResp.Reason != "" {
			result.LLMWarnings = append(result.LLMWarnings, llmResp.Reason)
		}
	}
}
