package service

import (
	"testing"

	"taskflow/internal/model"
)

func TestStateMachine_CanTransition(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		name     string
		from     model.TaskStatus
		to       model.TaskStatus
		expected bool
	}{
		{"PENDING -> RUNNING", model.TaskStatusPending, model.TaskStatusRunning, true},
		{"PENDING -> CANCELLED", model.TaskStatusPending, model.TaskStatusCancelled, true},
		{"PENDING -> SUCCEEDED", model.TaskStatusPending, model.TaskStatusSucceeded, false},
		{"PENDING -> FAILED", model.TaskStatusPending, model.TaskStatusFailed, false},

		{"RUNNING -> SUCCEEDED", model.TaskStatusRunning, model.TaskStatusSucceeded, true},
		{"RUNNING -> FAILED", model.TaskStatusRunning, model.TaskStatusFailed, true},
		{"RUNNING -> TIMEOUT", model.TaskStatusRunning, model.TaskStatusTimeout, true},
		{"RUNNING -> CANCELLED", model.TaskStatusRunning, model.TaskStatusCancelled, true},
		{"RUNNING -> PENDING", model.TaskStatusRunning, model.TaskStatusPending, false},

		{"FAILED -> PENDING", model.TaskStatusFailed, model.TaskStatusPending, true},
		{"FAILED -> CANCELLED", model.TaskStatusFailed, model.TaskStatusCancelled, true},
		{"FAILED -> RUNNING", model.TaskStatusFailed, model.TaskStatusRunning, false},

		{"SUCCEEDED -> any", model.TaskStatusSucceeded, model.TaskStatusPending, false},
		{"CANCELLED -> any", model.TaskStatusCancelled, model.TaskStatusPending, false},
		{"TIMEOUT -> any", model.TaskStatusTimeout, model.TaskStatusPending, false},

		{"UNSPECIFIED -> PENDING", model.TaskStatusUnspecified, model.TaskStatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sm.CanTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransition(%s, %s) = %v, expected %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestStateMachine_Transition(t *testing.T) {
	sm := NewStateMachine()

	task := &model.Task{
		ID:     "test-1",
		Status: model.TaskStatusPending,
		Name:   "Test Task",
	}

	// 测试有效的状态转换
	err := sm.Transition(task, model.TaskStatusRunning, "test-operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != model.TaskStatusRunning {
		t.Errorf("expected status RUNNING, got %v", task.Status)
	}
	if task.StartedAt == nil {
		t.Error("StartedAt should be set")
	}

	// 测试无效的状态转换
	task2 := &model.Task{
		ID:     "test-2",
		Status: model.TaskStatusSucceeded,
		Name:   "Test Task 2",
	}
	err = sm.Transition(task2, model.TaskStatusPending, "test-operator")
	if err == nil {
		t.Error("expected error for invalid transition from SUCCEEDED to PENDING")
	}
}

func TestStateMachine_TransitionHooks(t *testing.T) {
	sm := NewStateMachine()

	// 测试 RUNNING -> SUCCEEDED 转换
	task := &model.Task{
		ID:     "test-3",
		Status: model.TaskStatusRunning,
		Name:   "Test Task 3",
	}
	err := sm.Transition(task, model.TaskStatusSucceeded, "test-operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
	if task.ErrorMessage != "" {
		t.Error("ErrorMessage should be cleared on success")
	}

	// 测试 RUNNING -> FAILED 转换
	task2 := &model.Task{
		ID:     "test-4",
		Status: model.TaskStatusRunning,
		Name:   "Test Task 4",
	}
	err = sm.Transition(task2, model.TaskStatusFailed, "test-operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task2.RetryCount != 1 {
		t.Errorf("expected RetryCount = 1, got %d", task2.RetryCount)
	}

	// 测试 PENDING -> CANCELLED 转换
	task3 := &model.Task{
		ID:     "test-5",
		Status: model.TaskStatusPending,
		Name:   "Test Task 5",
	}
	err = sm.Transition(task3, model.TaskStatusCancelled, "test-operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStateMachine_GetAllowedTransitions(t *testing.T) {
	sm := NewStateMachine()

	// PENDING 允许的转换
	pendingTransitions := sm.GetAllowedTransitions(model.TaskStatusPending)
	if len(pendingTransitions) != 2 {
		t.Errorf("expected 2 allowed transitions from PENDING, got %d", len(pendingTransitions))
	}

	// RUNNING 允许的转换
	runningTransitions := sm.GetAllowedTransitions(model.TaskStatusRunning)
	if len(runningTransitions) != 4 {
		t.Errorf("expected 4 allowed transitions from RUNNING, got %d", len(runningTransitions))
	}

	// 终态不允许转换
	succeededTransitions := sm.GetAllowedTransitions(model.TaskStatusSucceeded)
	if len(succeededTransitions) != 0 {
		t.Errorf("expected 0 allowed transitions from SUCCEEDED, got %d", len(succeededTransitions))
	}
}

func TestStateMachine_IsTerminal(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		status   model.TaskStatus
		expected bool
	}{
		{model.TaskStatusUnspecified, false},
		{model.TaskStatusPending, false},
		{model.TaskStatusRunning, false},
		{model.TaskStatusSucceeded, true},
		{model.TaskStatusFailed, true},
		{model.TaskStatusCancelled, true},
		{model.TaskStatusTimeout, true},
	}

	for _, tt := range tests {
		result := sm.IsTerminal(tt.status)
		if result != tt.expected {
			t.Errorf("IsTerminal(%s) = %v, expected %v", tt.status, result, tt.expected)
		}
	}
}
