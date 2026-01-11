package plugin

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// TestNewRunManager tests the RunManager constructor
func TestNewRunManager(t *testing.T) {
	rm := NewRunManager(backend.Logger)

	if rm == nil {
		t.Fatal("expected non-nil RunManager")
	}

	if rm.runs == nil {
		t.Error("expected runs map to be initialized")
	}

	if rm.cleanupTicker == nil {
		t.Error("expected cleanup ticker to be initialized")
	}

	if rm.stopCleanup == nil {
		t.Error("expected stopCleanup channel to be initialized")
	}

	// Cleanup
	rm.Stop()
}

// TestCreateRun tests run creation
func TestCreateRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	tests := []struct {
		name           string
		conversationID string
		userLogin      string
		wantErr        bool
	}{
		{
			name:           "valid run creation",
			conversationID: "conv-123",
			userLogin:      "testuser",
			wantErr:        false,
		},
		{
			name:           "empty conversation ID",
			conversationID: "",
			userLogin:      "testuser",
			wantErr:        false, // No validation in CreateRun
		},
		{
			name:           "empty user login",
			conversationID: "conv-123",
			userLogin:      "",
			wantErr:        false, // No validation in CreateRun
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run, err := rm.CreateRun(context.Background(), tt.conversationID, tt.userLogin)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if run.RunID == "" {
					t.Error("expected non-empty RunID")
				}

				if run.ConversationID != tt.conversationID {
					t.Errorf("expected conversationID %s, got %s", tt.conversationID, run.ConversationID)
				}

				if run.UserLogin != tt.userLogin {
					t.Errorf("expected userLogin %s, got %s", tt.userLogin, run.UserLogin)
				}

				if run.Status != RunStatusPending {
					t.Errorf("expected status %s, got %s", RunStatusPending, run.Status)
				}

				if run.EventChan == nil {
					t.Error("expected event channel to be initialized")
				}

				if run.CancelCtx == nil {
					t.Error("expected cancel context to be initialized")
				}

				if run.CancelFunc == nil {
					t.Error("expected cancel func to be initialized")
				}

				// Cleanup
				rm.CleanupRun(run.RunID)
			}
		})
	}
}

// TestGetRun tests run retrieval
func TestGetRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	// Create a run
	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	tests := []struct {
		name    string
		runID   string
		wantErr bool
	}{
		{
			name:    "existing run",
			runID:   run.RunID,
			wantErr: false,
		},
		{
			name:    "non-existent run",
			runID:   "nonexistent-run",
			wantErr: true,
		},
		{
			name:    "empty run ID",
			runID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rm.GetRun(tt.runID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("expected non-nil run")
			}

			if !tt.wantErr && got.RunID != tt.runID {
				t.Errorf("expected runID %s, got %s", tt.runID, got.RunID)
			}
		})
	}
}

// TestUpdateRunStatus tests status updates
func TestUpdateRunStatus(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	tests := []struct {
		name      string
		runID     string
		newStatus RunStatus
		wantErr   bool
	}{
		{
			name:      "valid status update",
			runID:     run.RunID,
			newStatus: RunStatusExecuting,
			wantErr:   false,
		},
		{
			name:      "update to completed",
			runID:     run.RunID,
			newStatus: RunStatusCompleted,
			wantErr:   false,
		},
		{
			name:      "update non-existent run",
			runID:     "nonexistent-run",
			newStatus: RunStatusExecuting,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.UpdateRunStatus(tt.runID, tt.newStatus)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateRunStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updatedRun, _ := rm.GetRun(tt.runID)
				updatedRun.mu.RLock()
				status := updatedRun.Status
				updatedRun.mu.RUnlock()

				if status != tt.newStatus {
					t.Errorf("expected status %s, got %s", tt.newStatus, status)
				}
			}
		})
	}
}

// TestSetPlan tests execution plan setting
func TestSetPlan(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	plan := &ExecutionPlan{
		Goal: "Test goal",
		Steps: []PlannedStep{
			{Index: 0, Title: "Step 1", Description: "First step", Status: "pending"},
			{Index: 1, Title: "Step 2", Description: "Second step", Status: "pending"},
		},
		EstimatedDuration: "5m",
	}

	tests := []struct {
		name    string
		runID   string
		plan    *ExecutionPlan
		wantErr bool
	}{
		{
			name:    "valid plan",
			runID:   run.RunID,
			plan:    plan,
			wantErr: false,
		},
		{
			name:    "non-existent run",
			runID:   "nonexistent-run",
			plan:    plan,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.SetPlan(tt.runID, tt.plan)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.plan != nil {
				updatedRun, _ := rm.GetRun(tt.runID)
				updatedRun.mu.RLock()
				gotPlan := updatedRun.Plan
				updatedRun.mu.RUnlock()

				if gotPlan == nil {
					t.Error("expected plan to be set")
				}

				if gotPlan != nil && gotPlan.Goal != tt.plan.Goal {
					t.Errorf("expected goal %s, got %s", tt.plan.Goal, gotPlan.Goal)
				}

				if gotPlan != nil && len(gotPlan.Steps) != len(tt.plan.Steps) {
					t.Errorf("expected %d steps, got %d", len(tt.plan.Steps), len(gotPlan.Steps))
				}
			}
		})
	}
}

// TestAddArtifact tests artifact addition
func TestAddArtifact(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	artifact := Artifact{
		ID:       "artifact-1",
		Type:     "query_result",
		Content:  "test content",
		Metadata: map[string]interface{}{"key": "value"},
	}

	tests := []struct {
		name     string
		runID    string
		artifact Artifact
		wantErr  bool
	}{
		{
			name:     "valid artifact",
			runID:    run.RunID,
			artifact: artifact,
			wantErr:  false,
		},
		{
			name:     "non-existent run",
			runID:    "nonexistent-run",
			artifact: artifact,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.AddArtifact(tt.runID, tt.artifact)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddArtifact() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updatedRun, _ := rm.GetRun(tt.runID)
				updatedRun.mu.RLock()
				artifacts := updatedRun.Artifacts
				updatedRun.mu.RUnlock()

				if len(artifacts) == 0 {
					t.Error("expected artifact to be added")
				}

				found := false
				for _, a := range artifacts {
					if a.ID == tt.artifact.ID {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("artifact %s not found in run", tt.artifact.ID)
				}
			}
		})
	}
}

// TestUpdateStepStatus tests step status updates
func TestUpdateStepStatus(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	plan := &ExecutionPlan{
		Goal: "Test goal",
		Steps: []PlannedStep{
			{Index: 0, Title: "Step 1", Status: "pending"},
			{Index: 1, Title: "Step 2", Status: "pending"},
		},
	}
	rm.SetPlan(run.RunID, plan)

	tests := []struct {
		name      string
		runID     string
		stepIndex int
		status    string
		wantErr   bool
	}{
		{
			name:      "valid step update",
			runID:     run.RunID,
			stepIndex: 0,
			status:    "in_progress",
			wantErr:   false,
		},
		{
			name:      "update second step",
			runID:     run.RunID,
			stepIndex: 1,
			status:    "completed",
			wantErr:   false,
		},
		{
			name:      "invalid step index",
			runID:     run.RunID,
			stepIndex: 10,
			status:    "in_progress",
			wantErr:   true,
		},
		{
			name:      "non-existent run",
			runID:     "nonexistent-run",
			stepIndex: 0,
			status:    "in_progress",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.UpdateStepStatus(tt.runID, tt.stepIndex, tt.status)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStepStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updatedRun, _ := rm.GetRun(tt.runID)
				updatedRun.mu.RLock()
				stepStatus := updatedRun.Plan.Steps[tt.stepIndex].Status
				currentStepIndex := updatedRun.CurrentStepIndex
				updatedRun.mu.RUnlock()

				if stepStatus != tt.status {
					t.Errorf("expected step status %s, got %s", tt.status, stepStatus)
				}

				if currentStepIndex != tt.stepIndex {
					t.Errorf("expected currentStepIndex %d, got %d", tt.stepIndex, currentStepIndex)
				}
			}
		})
	}
}

// TestPauseRun tests run pausing
func TestPauseRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	tests := []struct {
		name          string
		runID         string
		initialStatus RunStatus
		wantErr       bool
	}{
		{
			name:          "pause executing run",
			runID:         run.RunID,
			initialStatus: RunStatusExecuting,
			wantErr:       false,
		},
		{
			name:          "pause pending run - should fail",
			runID:         run.RunID,
			initialStatus: RunStatusPending,
			wantErr:       true,
		},
		{
			name:          "pause completed run - should fail",
			runID:         run.RunID,
			initialStatus: RunStatusCompleted,
			wantErr:       true,
		},
		{
			name:          "non-existent run",
			runID:         "nonexistent-run",
			initialStatus: RunStatusExecuting,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial status
			if tt.runID != "nonexistent-run" {
				rm.UpdateRunStatus(tt.runID, tt.initialStatus)
			}

			err := rm.PauseRun(tt.runID)

			if (err != nil) != tt.wantErr {
				t.Errorf("PauseRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updatedRun, _ := rm.GetRun(tt.runID)
				updatedRun.mu.RLock()
				status := updatedRun.Status
				updatedRun.mu.RUnlock()

				if status != RunStatusPaused {
					t.Errorf("expected status %s, got %s", RunStatusPaused, status)
				}
			}
		})
	}
}

// TestResumeRun tests run resuming
func TestResumeRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	tests := []struct {
		name          string
		runID         string
		initialStatus RunStatus
		wantErr       bool
	}{
		{
			name:          "resume paused run",
			runID:         run.RunID,
			initialStatus: RunStatusPaused,
			wantErr:       false,
		},
		{
			name:          "resume executing run - should fail",
			runID:         run.RunID,
			initialStatus: RunStatusExecuting,
			wantErr:       true,
		},
		{
			name:          "non-existent run",
			runID:         "nonexistent-run",
			initialStatus: RunStatusPaused,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial status
			if tt.runID != "nonexistent-run" {
				rm.UpdateRunStatus(tt.runID, tt.initialStatus)
			}

			err := rm.ResumeRun(tt.runID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResumeRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updatedRun, _ := rm.GetRun(tt.runID)
				updatedRun.mu.RLock()
				status := updatedRun.Status
				updatedRun.mu.RUnlock()

				if status != RunStatusExecuting {
					t.Errorf("expected status %s, got %s", RunStatusExecuting, status)
				}
			}
		})
	}
}

// TestCancelRun tests run cancellation
func TestCancelRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	tests := []struct {
		name          string
		initialStatus RunStatus
		wantErr       bool
	}{
		{
			name:          "cancel pending run",
			initialStatus: RunStatusPending,
			wantErr:       false,
		},
		{
			name:          "cancel executing run",
			initialStatus: RunStatusExecuting,
			wantErr:       false,
		},
		{
			name:          "cancel paused run",
			initialStatus: RunStatusPaused,
			wantErr:       false,
		},
		{
			name:          "cancel completed run - should fail",
			initialStatus: RunStatusCompleted,
			wantErr:       true,
		},
		{
			name:          "cancel already cancelled run - should fail",
			initialStatus: RunStatusCancelled,
			wantErr:       true,
		},
		{
			name:          "cancel failed run - should fail",
			initialStatus: RunStatusFailed,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
			rm.UpdateRunStatus(run.RunID, tt.initialStatus)

			err := rm.CancelRun(run.RunID)

			if (err != nil) != tt.wantErr {
				t.Errorf("CancelRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updatedRun, _ := rm.GetRun(run.RunID)
				updatedRun.mu.RLock()
				status := updatedRun.Status
				updatedRun.mu.RUnlock()

				if status != RunStatusCancelled {
					t.Errorf("expected status %s, got %s", RunStatusCancelled, status)
				}

				// Verify context was cancelled
				select {
				case <-updatedRun.CancelCtx.Done():
					// Expected
				default:
					t.Error("expected context to be cancelled")
				}
			}

			// Cleanup
			rm.CleanupRun(run.RunID)
		})
	}
}

// TestCheckPauseOrCancel tests pause/cancel checking
func TestCheckPauseOrCancel(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	t.Run("executing run - should continue", func(t *testing.T) {
		run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
		rm.UpdateRunStatus(run.RunID, RunStatusExecuting)

		shouldContinue, wasPaused := rm.CheckPauseOrCancel(run.RunID)

		if !shouldContinue {
			t.Error("expected run to continue")
		}

		if wasPaused {
			t.Error("expected run not to be paused")
		}

		rm.CleanupRun(run.RunID)
	})

	t.Run("cancelled run - should not continue", func(t *testing.T) {
		run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
		run.CancelFunc()

		shouldContinue, wasPaused := rm.CheckPauseOrCancel(run.RunID)

		if shouldContinue {
			t.Error("expected run not to continue")
		}

		if wasPaused {
			t.Error("expected run not to be paused")
		}

		rm.CleanupRun(run.RunID)
	})

	t.Run("non-existent run", func(t *testing.T) {
		shouldContinue, wasPaused := rm.CheckPauseOrCancel("nonexistent-run")

		if shouldContinue {
			t.Error("expected non-existent run not to continue")
		}

		if wasPaused {
			t.Error("expected non-existent run not to be paused")
		}
	})
}

// TestCloseEventChannel tests event channel closing
func TestCloseEventChannel(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	err := rm.CloseEventChannel(run.RunID)
	if err != nil {
		t.Errorf("CloseEventChannel() error = %v", err)
	}

	// Verify channel is closed by trying to receive
	select {
	case _, ok := <-run.EventChan:
		if ok {
			t.Error("expected channel to be closed")
		}
	default:
		t.Error("expected channel to be closed")
	}

	rm.CleanupRun(run.RunID)
}

// TestCleanupRun tests manual run cleanup
func TestCleanupRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	err := rm.CleanupRun(run.RunID)
	if err != nil {
		t.Errorf("CleanupRun() error = %v", err)
	}

	// Verify run is removed
	_, err = rm.GetRun(run.RunID)
	if err == nil {
		t.Error("expected run to be removed")
	}

	// Verify context was cancelled
	select {
	case <-run.CancelCtx.Done():
		// Expected
	default:
		t.Error("expected context to be cancelled")
	}
}

// TestGetRunCount tests run counting
func TestGetRunCount(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	initialCount := rm.GetRunCount()

	// Create runs
	run1, _ := rm.CreateRun(context.Background(), "conv-1", "user1")
	run2, _ := rm.CreateRun(context.Background(), "conv-2", "user2")
	run3, _ := rm.CreateRun(context.Background(), "conv-3", "user1")

	count := rm.GetRunCount()
	expectedCount := initialCount + 3

	if count != expectedCount {
		t.Errorf("expected count %d, got %d", expectedCount, count)
	}

	// Cleanup one run
	rm.CleanupRun(run1.RunID)

	count = rm.GetRunCount()
	expectedCount = initialCount + 2

	if count != expectedCount {
		t.Errorf("expected count %d after cleanup, got %d", expectedCount, count)
	}

	// Cleanup remaining
	rm.CleanupRun(run2.RunID)
	rm.CleanupRun(run3.RunID)
}

// TestGetUserRunCount tests per-user run counting
func TestGetUserRunCount(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	// Create runs for different users
	run1, _ := rm.CreateRun(context.Background(), "conv-1", "user1")
	run2, _ := rm.CreateRun(context.Background(), "conv-2", "user1")
	run3, _ := rm.CreateRun(context.Background(), "conv-3", "user2")

	user1Count := rm.GetUserRunCount("user1")
	if user1Count != 2 {
		t.Errorf("expected user1 count 2, got %d", user1Count)
	}

	user2Count := rm.GetUserRunCount("user2")
	if user2Count != 1 {
		t.Errorf("expected user2 count 1, got %d", user2Count)
	}

	user3Count := rm.GetUserRunCount("user3")
	if user3Count != 0 {
		t.Errorf("expected user3 count 0, got %d", user3Count)
	}

	// Cleanup
	rm.CleanupRun(run1.RunID)
	rm.CleanupRun(run2.RunID)
	rm.CleanupRun(run3.RunID)
}

// TestCleanupOldRuns tests automatic cleanup
func TestCleanupOldRuns(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	// Create completed run with old timestamp
	run1, _ := rm.CreateRun(context.Background(), "conv-1", "user1")
	rm.UpdateRunStatus(run1.RunID, RunStatusCompleted)

	// Manually set old timestamp
	run1.mu.Lock()
	run1.UpdatedAt = time.Now().UTC().Add(-2 * time.Hour)
	run1.mu.Unlock()

	// Create recent run
	run2, _ := rm.CreateRun(context.Background(), "conv-2", "user2")
	rm.UpdateRunStatus(run2.RunID, RunStatusCompleted)

	initialCount := rm.GetRunCount()

	// Run cleanup
	rm.cleanupOldRuns()

	finalCount := rm.GetRunCount()

	// Old run should be cleaned up
	if finalCount != initialCount-1 {
		t.Errorf("expected count to decrease by 1, got initial=%d final=%d", initialCount, finalCount)
	}

	// Recent run should still exist
	_, err := rm.GetRun(run2.RunID)
	if err != nil {
		t.Error("expected recent run to still exist")
	}

	// Old run should be gone
	_, err = rm.GetRun(run1.RunID)
	if err == nil {
		t.Error("expected old run to be cleaned up")
	}

	// Cleanup
	rm.CleanupRun(run2.RunID)
}

// TestStop tests manager shutdown
func TestStop(t *testing.T) {
	rm := NewRunManager(backend.Logger)

	// Create some runs
	run1, _ := rm.CreateRun(context.Background(), "conv-1", "user1")
	run2, _ := rm.CreateRun(context.Background(), "conv-2", "user2")

	// Stop manager
	rm.Stop()

	// Verify contexts are cancelled
	select {
	case <-run1.CancelCtx.Done():
		// Expected
	default:
		t.Error("expected run1 context to be cancelled")
	}

	select {
	case <-run2.CancelCtx.Done():
		// Expected
	default:
		t.Error("expected run2 context to be cancelled")
	}
}

// TestGenerateRunID tests run ID generation
func TestGenerateRunID(t *testing.T) {
	id1 := generateRunID()
	id2 := generateRunID()

	if id1 == "" {
		t.Error("expected non-empty run ID")
	}

	if id2 == "" {
		t.Error("expected non-empty run ID")
	}

	if id1 == id2 {
		t.Error("expected unique run IDs")
	}

	if len(id1) < 10 {
		t.Error("expected run ID to have reasonable length")
	}

	// Check format
	if id1[:4] != "run_" {
		t.Errorf("expected run ID to start with 'run_', got %s", id1[:4])
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := rm.CreateRun(context.Background(), "conv-test", "testuser")
			if err != nil {
				t.Errorf("CreateRun() error in goroutine %d: %v", index, err)
			}
		}(i)
	}

	wg.Wait()

	count := rm.GetRunCount()
	if count < numGoroutines {
		t.Errorf("expected at least %d runs, got %d", numGoroutines, count)
	}

	// Concurrent reads
	rm.mu.RLock()
	runIDs := make([]string, 0, len(rm.runs))
	for runID := range rm.runs {
		runIDs = append(runIDs, runID)
	}
	rm.mu.RUnlock()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for _, runID := range runIDs {
				_, err := rm.GetRun(runID)
				if err != nil {
					t.Errorf("GetRun() error in goroutine %d: %v", index, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Cleanup all
	for _, runID := range runIDs {
		rm.CleanupRun(runID)
	}
}

// TestUpdateStepStatusNoPlan tests updating step when no plan exists
func TestUpdateStepStatusNoPlan(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	err := rm.UpdateStepStatus(run.RunID, 0, "in_progress")
	if err == nil {
		t.Error("expected error when updating step with no plan")
	}

	rm.CleanupRun(run.RunID)
}

// TestScheduleCleanup tests scheduled cleanup functionality
func TestScheduleCleanup(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")

	// Schedule cleanup with short delay for testing
	delay := 100 * time.Millisecond
	rm.ScheduleCleanup(run.RunID, delay)

	// Verify cleanup is scheduled
	run, err := rm.GetRun(run.RunID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	run.mu.RLock()
	status := run.CleanupStatus
	scheduledAt := run.CleanupScheduledAt
	run.mu.RUnlock()

	if status != "scheduled" {
		t.Errorf("expected cleanup status 'scheduled', got %s", status)
	}

	if scheduledAt.IsZero() {
		t.Error("expected CleanupScheduledAt to be set")
	}

	// Wait for cleanup to execute
	time.Sleep(delay + 50*time.Millisecond)

	// Verify run is deleted
	_, err = rm.GetRun(run.RunID)
	if err == nil {
		t.Error("expected run to be deleted after cleanup")
	}
}

// TestScheduleCleanupNonExistentRun tests scheduling cleanup for non-existent run
func TestScheduleCleanupNonExistentRun(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	// This should not panic, just log error
	rm.ScheduleCleanup("nonexistent-run", 1*time.Hour)

	// Test passes if no panic occurs
}

// TestScheduleCleanupStatusTracking tests cleanup status transitions
func TestScheduleCleanupStatusTracking(t *testing.T) {
	rm := NewRunManager(backend.Logger)
	defer rm.Stop()

	run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
	runID := run.RunID

	// Schedule cleanup with short delay
	delay := 50 * time.Millisecond
	rm.ScheduleCleanup(runID, delay)

	// Check status is "scheduled"
	run, _ = rm.GetRun(runID)
	run.mu.RLock()
	status := run.CleanupStatus
	run.mu.RUnlock()

	if status != "scheduled" {
		t.Errorf("expected status 'scheduled', got %s", status)
	}

	// Wait for cleanup to start
	time.Sleep(delay + 10*time.Millisecond)

	// Run should be deleted by now (successful cleanup)
	_, err := rm.GetRun(runID)
	if err == nil {
		t.Error("expected run to be deleted")
	}
}
