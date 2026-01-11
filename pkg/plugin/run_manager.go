package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type RunStatus string

const (
	RunStatusPending    RunStatus = "pending"
	RunStatusPlanning   RunStatus = "planning"
	RunStatusExecuting  RunStatus = "executing"
	RunStatusPaused     RunStatus = "paused"
	RunStatusCompleted  RunStatus = "completed"
	RunStatusCancelled  RunStatus = "cancelled"
	RunStatusFailed     RunStatus = "failed"
)

type RunState struct {
	RunID              string
	ConversationID     string
	UserLogin          string
	Status             RunStatus
	Plan               *ExecutionPlan
	CurrentStepIndex   int
	Artifacts          []Artifact
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CleanupScheduledAt time.Time
	CleanupError       error
	CleanupStatus      string // "scheduled", "cleaning", "cleaned", "failed"
	CancelCtx          context.Context
	CancelFunc         context.CancelFunc
	EventChan          chan RunEvent
	pauseChan          chan bool
	mu                 sync.RWMutex
}

type ExecutionPlan struct {
	Goal              string        `json:"goal"`
	Steps             []PlannedStep `json:"steps"`
	EstimatedDuration string        `json:"estimatedDuration"`
}

type PlannedStep struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` 
}

type Artifact struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` 
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

type RunManager struct {
	runs          map[string]*RunState
	mu            sync.RWMutex
	logger        log.Logger
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

func NewRunManager(logger log.Logger) *RunManager {
	rm := &RunManager{
		runs:          make(map[string]*RunState),
		logger:        logger,
		cleanupTicker: time.NewTicker(10 * time.Minute),
		stopCleanup:   make(chan bool),
	}

	go rm.cleanupRoutine()

	return rm
}

func (rm *RunManager) CreateRun(sourceCtx context.Context, conversationID, userLogin string) (*RunState, error) {
	runID := generateRunID()

	detachedCtx := context.WithoutCancel(sourceCtx)

	ctx, cancel := context.WithCancel(detachedCtx)

	run := &RunState{
		RunID:          runID,
		ConversationID: conversationID,
		UserLogin:      userLogin,
		Status:         RunStatusPending,
		Artifacts:      []Artifact{},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		CancelCtx:      ctx,
		CancelFunc:     cancel,
		EventChan:      make(chan RunEvent, 100), 
		pauseChan:      make(chan bool, 1),
	}

	rm.mu.Lock()
	rm.runs[runID] = run
	rm.mu.Unlock()

	rm.logger.Info("Run created", "runId", runID, "conversationId", conversationID, "user", userLogin)

	return run, nil
}

func (rm *RunManager) GetRun(runID string) (*RunState, error) {
	rm.mu.RLock()
	run, exists := rm.runs[runID]
	rm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	return run, nil
}

func (rm *RunManager) UpdateRunStatus(runID string, status RunStatus) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	run.Status = status
	run.UpdatedAt = time.Now().UTC()
	run.mu.Unlock()

	rm.logger.Debug("Run status updated", "runId", runID, "status", status)

	return nil
}

func (rm *RunManager) SetPlan(runID string, plan *ExecutionPlan) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	run.Plan = plan
	run.UpdatedAt = time.Now().UTC()
	run.mu.Unlock()

	rm.logger.Info("Execution plan set", "runId", runID, "stepCount", len(plan.Steps))

	return nil
}

func (rm *RunManager) AddArtifact(runID string, artifact Artifact) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	run.Artifacts = append(run.Artifacts, artifact)
	run.UpdatedAt = time.Now().UTC()
	run.mu.Unlock()

	rm.logger.Debug("Artifact added", "runId", runID, "artifactId", artifact.ID, "type", artifact.Type)

	return nil
}

func (rm *RunManager) UpdateStepStatus(runID string, stepIndex int, status string) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	defer run.mu.Unlock()

	if run.Plan == nil || stepIndex >= len(run.Plan.Steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	run.Plan.Steps[stepIndex].Status = status
	run.CurrentStepIndex = stepIndex
	run.UpdatedAt = time.Now().UTC()

	return nil
}

func (rm *RunManager) PauseRun(runID string) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	currentStatus := run.Status
	run.mu.Unlock()

	if currentStatus != RunStatusExecuting {
		return fmt.Errorf("cannot pause run in status: %s", currentStatus)
	}

	if err := rm.UpdateRunStatus(runID, RunStatusPaused); err != nil {
		return err
	}

	rm.logger.Info("Run paused", "runId", runID)

	return nil
}

func (rm *RunManager) ResumeRun(runID string) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	currentStatus := run.Status
	run.mu.Unlock()

	if currentStatus != RunStatusPaused {
		return fmt.Errorf("cannot resume run in status: %s", currentStatus)
	}

	if err := rm.UpdateRunStatus(runID, RunStatusExecuting); err != nil {
		return err
	}

	select {
	case run.pauseChan <- true:
	default:
	}

	rm.logger.Info("Run resumed", "runId", runID)

	return nil
}

func (rm *RunManager) CancelRun(runID string) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	run.mu.Lock()
	currentStatus := run.Status
	run.mu.Unlock()

	if currentStatus == RunStatusCompleted || currentStatus == RunStatusCancelled || currentStatus == RunStatusFailed {
		return fmt.Errorf("cannot cancel run in status: %s", currentStatus)
	}

	run.CancelFunc()

	if err := rm.UpdateRunStatus(runID, RunStatusCancelled); err != nil {
		return err
	}

	rm.logger.Info("Run cancelled", "runId", runID)

	return nil
}

func (rm *RunManager) CheckPauseOrCancel(runID string) (bool, bool) {
	run, err := rm.GetRun(runID)
	if err != nil {
		return false, false
	}

	select {
	case <-run.CancelCtx.Done():
		return false, false
	default:
	}

	run.mu.RLock()
	status := run.Status
	run.mu.RUnlock()

	if status == RunStatusPaused {
		select {
		case <-run.pauseChan:
			return true, true
		case <-run.CancelCtx.Done():
			return false, true
		case <-time.After(1 * time.Hour): 
			rm.logger.Warn("Pause timeout exceeded", "runId", runID)
			return false, true
		}
	}

	return true, false
}

func (rm *RunManager) CloseEventChannel(runID string) error {
	run, err := rm.GetRun(runID)
	if err != nil {
		return err
	}

	close(run.EventChan)
	rm.logger.Debug("Event channel closed", "runId", runID)

	return nil
}

func (rm *RunManager) CleanupRun(runID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	run, exists := rm.runs[runID]
	if !exists {
		return fmt.Errorf("run not found: %s", runID)
	}

	run.CancelFunc()

	delete(rm.runs, runID)

	rm.logger.Info("Run cleaned up", "runId", runID)

	return nil
}

func (rm *RunManager) ScheduleCleanup(runID string, delay time.Duration) {
	run, err := rm.GetRun(runID)
	if err != nil {
		rm.logger.Error("Failed to schedule cleanup", "runId", runID, "error", err)
		return
	}

	run.mu.Lock()
	run.CleanupScheduledAt = time.Now().UTC().Add(delay)
	run.CleanupStatus = "scheduled"
	run.mu.Unlock()

	go func() {
		time.Sleep(delay)

		run.mu.Lock()
		run.CleanupStatus = "cleaning"
		run.mu.Unlock()

		if err := rm.CleanupRun(runID); err != nil {
			run.mu.Lock()
			run.CleanupStatus = "failed"
			run.CleanupError = err
			run.mu.Unlock()

			rm.logger.Error("Run cleanup failed", "runId", runID, "error", err)
		} else {
			rm.logger.Info("Run cleanup completed", "runId", runID)
		}
	}()
}

func (rm *RunManager) cleanupRoutine() {
	for {
		select {
		case <-rm.cleanupTicker.C:
			rm.cleanupOldRuns()
		case <-rm.stopCleanup:
			rm.cleanupTicker.Stop()
			return
		}
	}
}

func (rm *RunManager) cleanupOldRuns() {
	now := time.Now().UTC()
	threshold := 1 * time.Hour

	rm.mu.Lock()
	defer rm.mu.Unlock()

	for runID, run := range rm.runs {
		run.mu.RLock()
		status := run.Status
		updatedAt := run.UpdatedAt
		run.mu.RUnlock()

		if (status == RunStatusCompleted || status == RunStatusCancelled || status == RunStatusFailed) &&
			now.Sub(updatedAt) > threshold {

			run.CancelFunc()

			delete(rm.runs, runID)

			rm.logger.Info("Run auto-cleaned", "runId", runID, "status", status, "age", now.Sub(updatedAt))
		}
	}
}

func (rm *RunManager) GetRunCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.runs)
}

func (rm *RunManager) GetUserRunCount(userLogin string) int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	count := 0
	for _, run := range rm.runs {
		run.mu.RLock()
		if run.UserLogin == userLogin {
			count++
		}
		run.mu.RUnlock()
	}

	return count
}

func (rm *RunManager) Stop() {
	rm.stopCleanup <- true

	rm.mu.Lock()
	for _, run := range rm.runs {
		run.CancelFunc()
	}
	rm.mu.Unlock()

	rm.logger.Info("RunManager stopped")
}

func generateRunID() string {
	return fmt.Sprintf("run_%s", uuid.New().String())
}
