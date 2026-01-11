package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type QueryValidationResult struct {
	Valid          bool
	Sanitized      bool
	SanitizedQuery string
	Error          error
	ViolationType  string 
	OriginalQuery  string

	LLMWarnings    []string 
	LLMSuggestions []string 
	LLMBlocked     bool     
}

type QueryValidator struct {
	settings *QueryValidationSettings
	app      *App 
}

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

func (v *QueryValidator) ValidateQuery(ctx context.Context, query string, dsType DatasourceType) *QueryValidationResult {
	if !v.settings.Enabled {
		return &QueryValidationResult{Valid: true}
	}

	var parserResult *QueryValidationResult
	switch dsType {
	case DatasourcePrometheus:
		if !v.settings.EnablePromQLValidation {
			return &QueryValidationResult{Valid: true}
		}
		parserResult = v.validatePromQL(query)
	case DatasourceLoki:
		if !v.settings.EnableLogQLValidation {
			return &QueryValidationResult{Valid: true}
		}
		parserResult = v.validateLogQL(query)
	case DatasourceTempo:
		if !v.settings.EnableTraceQLValidation {
			return &QueryValidationResult{Valid: true}
		}
		parserResult = v.validateTraceQL(query)
	default:
		parserResult = v.validateGeneric(query)
	}

	if !parserResult.Valid {
		return parserResult
	}

	if v.settings.EnableLLMValidation {
		llmResult := v.validateWithLLM(ctx, query, dsType, parserResult.SanitizedQuery)

		parserResult.LLMWarnings = llmResult.LLMWarnings
		parserResult.LLMSuggestions = llmResult.LLMSuggestions
		parserResult.LLMBlocked = llmResult.LLMBlocked

		if v.settings.LLMValidationMode == "strict" && llmResult.LLMBlocked {
			parserResult.Valid = false
			parserResult.Error = fmt.Errorf("query blocked by semantic validation: %s", strings.Join(llmResult.LLMWarnings, "; "))
			parserResult.ViolationType = "semantic"
		}
	}

	return parserResult
}

func (v *QueryValidator) validatePromQL(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true, 
	}

	genericResult := v.validateGeneric(query)
	if !genericResult.Valid {
		return genericResult
	}

	query = strings.TrimSpace(query)

	if len(query) == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("empty PromQL query")
		result.ViolationType = "syntax"
		return result
	}

	if !v.hasBalancedBraces(query) {
		if v.settings.StrictMode {
			result.Valid = false
			result.Error = fmt.Errorf("unbalanced braces in PromQL query")
			result.ViolationType = "syntax"
			return result
		}

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

	invalidOperatorPatterns := []string{
		"===",  
		"!==",  
		"<>",   
		"++",   
		"--",   
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

	complexity := v.countPromQLComplexityManual(query)
	if complexity > v.settings.MaxQueryComplexity {
		result.Valid = false
		result.Error = fmt.Errorf("query too complex: estimated %d nodes (max: %d)", complexity, v.settings.MaxQueryComplexity)
		result.ViolationType = "complexity"
		return result
	}

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

func (v *QueryValidator) validateLogQL(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true, 
	}

	genericResult := v.validateGeneric(query)
	if !genericResult.Valid {
		return genericResult
	}

	query = strings.TrimSpace(query)

	if len(query) == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("empty LogQL query")
		result.ViolationType = "syntax"
		return result
	}

	if !v.hasBalancedBraces(query) {
		if v.settings.StrictMode {
			result.Valid = false
			result.Error = fmt.Errorf("unbalanced braces in LogQL query")
			result.ViolationType = "syntax"
			return result
		}

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

	hasLogSelector := strings.Contains(query, "{") && strings.Contains(query, "}")

	if !hasLogSelector && v.settings.StrictMode {
		result.Valid = false
		result.Error = fmt.Errorf("LogQL query must contain a log stream selector {}")
		result.ViolationType = "syntax"
		return result
	}

	invalidOperatorPatterns := []string{
		"===",  
		"!==",  
		"<>",   
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

func (v *QueryValidator) validateTraceQL(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true, 
	}

	genericResult := v.validateGeneric(query)
	if !genericResult.Valid {
		return genericResult
	}

	query = strings.TrimSpace(query)

	if len(query) == 0 {
		result.Valid = false
		result.Error = fmt.Errorf("empty TraceQL query")
		result.ViolationType = "syntax"
		return result
	}

	if !v.hasBalancedBraces(query) {
		if v.settings.StrictMode {
			result.Valid = false
			result.Error = fmt.Errorf("unbalanced braces in TraceQL query")
			result.ViolationType = "syntax"
			return result
		}

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

	validPrefixes := []string{"span.", "resource.", "name", "duration", "status", "kind", "rootName", "rootServiceName"}
	hasValidPrefix := false

	if strings.Contains(query, "{") && strings.Contains(query, "}") {
		for _, prefix := range validPrefixes {
			if strings.Contains(query, prefix) {
				hasValidPrefix = true
				break
			}
		}

		if strings.Contains(query, ".") {
			hasValidPrefix = true
		}

		if strings.Contains(query, "{}") {
			hasValidPrefix = true
		}
	} else {
		hasValidPrefix = true
	}

	if !hasValidPrefix && v.settings.StrictMode {
		result.Valid = false
		result.Error = fmt.Errorf("TraceQL query must contain valid attribute selectors (span., resource., or intrinsic fields)")
		result.ViolationType = "syntax"
		return result
	}

	invalidOperatorPatterns := []string{
		"===",  
		"!==",  
		"<>",   
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

func (v *QueryValidator) hasBalancedBraces(query string) bool {
	counts := map[rune]int{
		'{': 0,
		'[': 0,
		'(': 0,
	}

	inString := false
	var stringDelim rune

	for i, ch := range query {
		if ch == '"' || ch == '\'' || ch == '`' {
			if !inString {
				inString = true
				stringDelim = ch
			} else if ch == stringDelim {
				if i > 0 && query[i-1] != '\\' {
					inString = false
				}
			}
			continue
		}

		if inString {
			continue
		}

		if ch == '{' || ch == '[' || ch == '(' {
			counts[ch]++
		}

		if ch == '}' {
			counts['{']--
		} else if ch == ']' {
			counts['[']--
		} else if ch == ')' {
			counts['(']--
		}
	}

	return counts['{'] == 0 && counts['['] == 0 && counts['('] == 0
}

func (v *QueryValidator) sanitizeTraceQL(query string) string {
	sanitized := strings.TrimSpace(query)


	return sanitized
}

func (v *QueryValidator) countTraceQLComplexity(query string) int {
	complexity := 1 

	complexity += strings.Count(query, "{")

	complexity += strings.Count(query, "&&")
	complexity += strings.Count(query, "||")

	complexity += strings.Count(query, "=")
	complexity += strings.Count(query, "!=")
	complexity += strings.Count(query, ">")
	complexity += strings.Count(query, "<")
	complexity += strings.Count(query, "=~")
	complexity += strings.Count(query, "!~")

	complexity += strings.Count(query, "|")

	complexity += strings.Count(query, "(")

	return complexity
}

func (v *QueryValidator) validateGeneric(query string) *QueryValidationResult {
	result := &QueryValidationResult{
		OriginalQuery: query,
		Valid:         true,
	}

	if len(query) > 10000 {
		result.Valid = false
		result.Error = fmt.Errorf("query too long: %d chars (max: 10000)", len(query))
		result.ViolationType = "length"
		return result
	}

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

func (v *QueryValidator) sanitizePromQL(query string) string {
	sanitized := strings.TrimSpace(query)


	return sanitized
}

func (v *QueryValidator) sanitizeLogQL(query string) string {
	sanitized := strings.TrimSpace(query)


	return sanitized
}

func (v *QueryValidator) countPromQLComplexityManual(query string) int {
	complexity := 1 

	complexity += strings.Count(query, "{")

	complexity += strings.Count(query, "and")
	complexity += strings.Count(query, "or")
	complexity += strings.Count(query, "unless")

	complexity += strings.Count(query, "==")
	complexity += strings.Count(query, "!=")
	complexity += strings.Count(query, ">")
	complexity += strings.Count(query, "<")
	complexity += strings.Count(query, "=~")
	complexity += strings.Count(query, "!~")

	complexity += strings.Count(query, "+")
	complexity += strings.Count(query, "-")
	complexity += strings.Count(query, "*")
	complexity += strings.Count(query, "/")
	complexity += strings.Count(query, "%")
	complexity += strings.Count(query, "^")

	complexity += strings.Count(query, "sum")
	complexity += strings.Count(query, "avg")
	complexity += strings.Count(query, "min")
	complexity += strings.Count(query, "max")
	complexity += strings.Count(query, "count")

	complexity += strings.Count(query, "(")

	complexity += strings.Count(query, "[")

	return complexity
}

func (v *QueryValidator) countLogQLComplexityManual(query string) int {
	complexity := 1 

	complexity += strings.Count(query, "{")

	complexity += strings.Count(query, "|=")
	complexity += strings.Count(query, "!=")
	complexity += strings.Count(query, "|~")
	complexity += strings.Count(query, "!~")

	complexity += strings.Count(query, "| json")
	complexity += strings.Count(query, "| logfmt")
	complexity += strings.Count(query, "| pattern")
	complexity += strings.Count(query, "| regexp")

	complexity += strings.Count(query, "sum")
	complexity += strings.Count(query, "avg")
	complexity += strings.Count(query, "min")
	complexity += strings.Count(query, "max")
	complexity += strings.Count(query, "count")
	complexity += strings.Count(query, "rate")

	complexity += strings.Count(query, "(")

	complexity += strings.Count(query, "[")

	return complexity
}

func (v *QueryValidator) checkPromQLFunctionsManual(query string) error {
	if len(v.settings.AllowedFunctions) == 0 {
		return nil
	}

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
	violationSet := make(map[string]bool) 

	for _, funcName := range commonFunctions {
		pattern := funcName + "("
		if strings.Contains(query, pattern) {
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

func (v *QueryValidator) validateWithLLM(ctx context.Context, originalQuery string, dsType DatasourceType, sanitizedQuery string) *QueryValidationResult {
	result := &QueryValidationResult{
		Valid:         true,
		OriginalQuery: originalQuery,
	}

	queryToValidate := originalQuery
	if sanitizedQuery != "" {
		queryToValidate = sanitizedQuery
	}

	prompt := v.buildLLMValidationPrompt(queryToValidate, dsType)

	llmResponse, err := v.callLLMForValidation(ctx, prompt)
	if err != nil {
		backend.Logger.Warn("LLM validation failed", "error", err)
		return result
	}

	v.parseLLMValidationResponse(llmResponse, result)

	return result
}

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

func (v *QueryValidator) callLLMForValidation(ctx context.Context, prompt string) (string, error) {
	// Check if LLM client is available
	if v.app.llmClient == nil {
		backend.Logger.Warn("LLM client not initialized, skipping semantic validation")
		return `{"safe": true, "warnings": ["LLM validation unavailable - no service account token configured"], "suggestions": [], "reason": ""}`, nil
	}

	// Check if settings are available
	if v.app.settings == nil {
		backend.Logger.Warn("Settings not initialized, skipping semantic validation")
		return `{"safe": true, "warnings": ["LLM validation unavailable - settings not initialized"], "suggestions": [], "reason": ""}`, nil
	}

	// Set aggressive timeout for validation (max 5 seconds)
	validationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	backend.Logger.Debug("Calling LLM for semantic query validation")

	// Prepare LLM request - use fast, cheap model for validation
	model := v.app.settings.LLMModel
	if model == "" {
		model = "gpt-4o-mini" // Default to fast, cheap model
	}

	llmRequest := LLMStreamRequest{
		Model: model,
		Messages: []AssistantMessage{
			{
				Role:    "system",
				Content: "You are a query validation expert. Respond only in valid JSON format.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.0,   // Deterministic responses
		MaxTokens:   500,   // Keep responses short
		Stream:      false, // Non-streaming
	}

	// Call LLM with timeout
	response, err := v.app.llmClient.SimpleChat(validationCtx, llmRequest)
	if err != nil {
		// FAIL-OPEN: On error, allow query but log warning
		backend.Logger.Warn("LLM validation call failed, allowing query",
			"error", err,
			"mode", v.settings.LLMValidationMode,
		)

		// Return safe response with warning
		return `{"safe": true, "warnings": ["LLM validation unavailable - service error"], "suggestions": [], "reason": ""}`, nil
	}

	backend.Logger.Debug("LLM validation completed", "responseLength", len(response))

	// Validate response is valid JSON
	var testJSON map[string]interface{}
	if err := json.Unmarshal([]byte(response), &testJSON); err != nil {
		backend.Logger.Warn("LLM returned invalid JSON, allowing query",
			"error", err,
			"response", response,
		)
		return `{"safe": true, "warnings": ["LLM validation returned invalid response"], "suggestions": [], "reason": ""}`, nil
	}

	return response, nil
}

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
