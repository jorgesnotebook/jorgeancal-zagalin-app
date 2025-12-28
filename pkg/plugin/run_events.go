package plugin

import "time"

// Event type constants
const (
	EventRunStarted       = "run_started"
	EventPlan             = "plan"
	EventStepStarted      = "step_started"
	EventProgress         = "progress"
	EventArtifact         = "artifact"
	EventAssistantDelta   = "assistant_delta"
	EventStepDone         = "step_done"
	EventAssistantMessage = "assistant_message"
	EventPaused           = "paused"
	EventResumed          = "resumed"
	EventCancelled        = "cancelled"
	EventFinal            = "final"
	EventError            = "error"
)

// RunEvent is the base event structure for SSE streaming
type RunEvent struct {
	Type      string      `json:"type"`
	RunID     string      `json:"runId"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// RunStartedEvent is emitted when a run begins
type RunStartedEvent struct {
	RunID          string `json:"runId"`
	ConversationID string `json:"conversationId"`
	Timestamp      string `json:"timestamp"`
}

// PlanEvent is emitted when the execution plan is generated
type PlanEvent struct {
	Goal              string        `json:"goal"`
	Steps             []PlannedStep `json:"steps"`
	EstimatedDuration string        `json:"estimatedDuration"`
}

// StepStartedEvent is emitted when a step begins execution
type StepStartedEvent struct {
	StepIndex   int    `json:"stepIndex"`
	StepTitle   string `json:"stepTitle"`
	Description string `json:"description"`
}

// ProgressEvent is emitted to show progress within a step
type ProgressEvent struct {
	StepIndex int    `json:"stepIndex"`
	Message   string `json:"message"`
}

// ArtifactEvent is emitted when an artifact is generated
type ArtifactEvent struct {
	ArtifactID string                 `json:"artifactId"`
	Type       string                 `json:"type"` // "query" | "link" | "trace_id" | "dashboard_link"
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	StepIndex  int                    `json:"stepIndex,omitempty"`
}

// AssistantDeltaEvent is emitted for streaming text chunks
type AssistantDeltaEvent struct {
	Delta string `json:"delta"`
}

// StepDoneEvent is emitted when a step completes
type StepDoneEvent struct {
	StepIndex  int    `json:"stepIndex"`
	StepTitle  string `json:"stepTitle"`
	Status     string `json:"status"` // "completed" | "failed"
	Conclusion string `json:"conclusion,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AssistantMessageEvent is emitted with the final assistant message
type AssistantMessageEvent struct {
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
	Text      string `json:"text"`
}

// PausedEvent is emitted when a run is paused
type PausedEvent struct {
	RunID string `json:"runId"`
}

// ResumedEvent is emitted when a run is resumed
type ResumedEvent struct {
	RunID string `json:"runId"`
}

// CancelledEvent is emitted when a run is cancelled
type CancelledEvent struct {
	RunID  string `json:"runId"`
	Reason string `json:"reason,omitempty"`
}

// FinalEvent is emitted when a run completes (success or failure)
type FinalEvent struct {
	Status         string `json:"status"` // "completed" | "cancelled" | "failed"
	TotalSteps     int    `json:"totalSteps"`
	CompletedSteps int    `json:"completedSteps"`
	FailedSteps    int    `json:"failedSteps"`
	TotalArtifacts int    `json:"totalArtifacts"`
	Duration       string `json:"duration"`
}

// ErrorEvent is emitted when an error occurs
type ErrorEvent struct {
	Message   string `json:"message"`
	StepIndex int    `json:"stepIndex,omitempty"`
	Retryable bool   `json:"retryable"`
}

// Helper functions to create events

// NewRunEvent creates a new RunEvent with the given type and data
func NewRunEvent(eventType, runID string, data interface{}) RunEvent {
	return RunEvent{
		Type:      eventType,
		RunID:     runID,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

// EmitRunStarted emits a run_started event
func EmitRunStarted(eventChan chan RunEvent, runID, conversationID string) {
	event := NewRunEvent(EventRunStarted, runID, RunStartedEvent{
		RunID:          runID,
		ConversationID: conversationID,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	})
	eventChan <- event
}

// EmitPlan emits a plan event
func EmitPlan(eventChan chan RunEvent, runID string, plan *ExecutionPlan) {
	event := NewRunEvent(EventPlan, runID, PlanEvent{
		Goal:              plan.Goal,
		Steps:             plan.Steps,
		EstimatedDuration: plan.EstimatedDuration,
	})
	eventChan <- event
}

// EmitStepStarted emits a step_started event
func EmitStepStarted(eventChan chan RunEvent, runID string, stepIndex int, title, description string) {
	event := NewRunEvent(EventStepStarted, runID, StepStartedEvent{
		StepIndex:   stepIndex,
		StepTitle:   title,
		Description: description,
	})
	eventChan <- event
}

// EmitProgress emits a progress event
func EmitProgress(eventChan chan RunEvent, runID string, stepIndex int, message string) {
	event := NewRunEvent(EventProgress, runID, ProgressEvent{
		StepIndex: stepIndex,
		Message:   message,
	})
	eventChan <- event
}

// EmitArtifact emits an artifact event
func EmitArtifact(eventChan chan RunEvent, runID string, artifact Artifact, stepIndex int) {
	event := NewRunEvent(EventArtifact, runID, ArtifactEvent{
		ArtifactID: artifact.ID,
		Type:       artifact.Type,
		Content:    artifact.Content,
		Metadata:   artifact.Metadata,
		StepIndex:  stepIndex,
	})
	eventChan <- event
}

// EmitAssistantDelta emits an assistant_delta event
func EmitAssistantDelta(eventChan chan RunEvent, runID, delta string) {
	event := NewRunEvent(EventAssistantDelta, runID, AssistantDeltaEvent{
		Delta: delta,
	})
	eventChan <- event
}

// EmitStepDone emits a step_done event
func EmitStepDone(eventChan chan RunEvent, runID string, stepIndex int, title, status, conclusion, errorMsg string) {
	event := NewRunEvent(EventStepDone, runID, StepDoneEvent{
		StepIndex:  stepIndex,
		StepTitle:  title,
		Status:     status,
		Conclusion: conclusion,
		Error:      errorMsg,
	})
	eventChan <- event
}

// EmitAssistantMessage emits an assistant_message event
func EmitAssistantMessage(eventChan chan RunEvent, runID, messageID, role, text string) {
	event := NewRunEvent(EventAssistantMessage, runID, AssistantMessageEvent{
		MessageID: messageID,
		Role:      role,
		Text:      text,
	})
	eventChan <- event
}

// EmitPaused emits a paused event
func EmitPaused(eventChan chan RunEvent, runID string) {
	event := NewRunEvent(EventPaused, runID, PausedEvent{
		RunID: runID,
	})
	eventChan <- event
}

// EmitResumed emits a resumed event
func EmitResumed(eventChan chan RunEvent, runID string) {
	event := NewRunEvent(EventResumed, runID, ResumedEvent{
		RunID: runID,
	})
	eventChan <- event
}

// EmitCancelled emits a cancelled event
func EmitCancelled(eventChan chan RunEvent, runID, reason string) {
	event := NewRunEvent(EventCancelled, runID, CancelledEvent{
		RunID:  runID,
		Reason: reason,
	})
	eventChan <- event
}

// EmitFinal emits a final event
func EmitFinal(eventChan chan RunEvent, runID, status string, totalSteps, completedSteps, failedSteps, totalArtifacts int, duration string) {
	event := NewRunEvent(EventFinal, runID, FinalEvent{
		Status:         status,
		TotalSteps:     totalSteps,
		CompletedSteps: completedSteps,
		FailedSteps:    failedSteps,
		TotalArtifacts: totalArtifacts,
		Duration:       duration,
	})
	eventChan <- event
}

// EmitError emits an error event
func EmitError(eventChan chan RunEvent, runID, message string, stepIndex int, retryable bool) {
	event := NewRunEvent(EventError, runID, ErrorEvent{
		Message:   message,
		StepIndex: stepIndex,
		Retryable: retryable,
	})
	eventChan <- event
}
