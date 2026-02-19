package model

import (
	"testing"
	"time"
)

func TestTaskStatus_String(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusUnspecified, "UNSPECIFIED"},
		{TaskStatusPending, "PENDING"},
		{TaskStatusRunning, "RUNNING"},
		{TaskStatusSucceeded, "SUCCEEDED"},
		{TaskStatusFailed, "FAILED"},
		{TaskStatusCancelled, "CANCELLED"},
		{TaskStatusTimeout, "TIMEOUT"},
	}

	for _, tt := range tests {
		result := tt.status.String()
		if result != tt.expected {
			t.Errorf("TaskStatus(%d).String() = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestTaskPriority_String(t *testing.T) {
	tests := []struct {
		priority TaskPriority
		expected string
	}{
		{TaskPriorityUnspecified, "UNSPECIFIED"},
		{TaskPriorityLow, "LOW"},
		{TaskPriorityNormal, "NORMAL"},
		{TaskPriorityHigh, "HIGH"},
		{TaskPriorityUrgent, "URGENT"},
	}

	for _, tt := range tests {
		result := tt.priority.String()
		if result != tt.expected {
			t.Errorf("TaskPriority(%d).String() = %s, expected %s", tt.priority, result, tt.expected)
		}
	}
}

func TestNewTask(t *testing.T) {
	inputParams := map[string]string{"key1": "value1", "key2": "value2"}
	dependencies := []string{"dep-1", "dep-2"}

	task := NewTask(
		"Test Task",
		"Test Description",
		TaskPriorityHigh,
		"test-type",
		inputParams,
		dependencies,
		3,
		"testuser",
	)

	if task.Name != "Test Task" {
		t.Errorf("expected Name 'Test Task', got '%s'", task.Name)
	}
	if task.Description != "Test Description" {
		t.Errorf("expected Description 'Test Description', got '%s'", task.Description)
	}
	if task.Priority != TaskPriorityHigh {
		t.Errorf("expected Priority HIGH, got %v", task.Priority)
	}
	if task.TaskType != "test-type" {
		t.Errorf("expected TaskType 'test-type', got '%s'", task.TaskType)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("expected Status PENDING, got %v", task.Status)
	}
	if task.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", task.MaxRetries)
	}
	if task.CreatedBy != "testuser" {
		t.Errorf("expected CreatedBy 'testuser', got '%s'", task.CreatedBy)
	}
	if task.InputParams["key1"] != "value1" {
		t.Errorf("expected InputParams[key1] 'value1', got '%s'", task.InputParams["key1"])
	}
	if len(task.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(task.Dependencies))
	}
	if task.ID != "" {
		t.Error("ID should be empty, to be set by caller")
	}
}

func TestTask_IsTerminal(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusUnspecified, false},
		{TaskStatusPending, false},
		{TaskStatusRunning, false},
		{TaskStatusSucceeded, true},
		{TaskStatusFailed, true},
		{TaskStatusCancelled, true},
		{TaskStatusTimeout, true},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		result := task.IsTerminal()
		if result != tt.expected {
			t.Errorf("Task.IsTerminal() with Status %s = %v, expected %v", tt.status, result, tt.expected)
		}
	}
}

func TestTask_CanRetry(t *testing.T) {
	tests := []struct {
		status     TaskStatus
		retryCount int32
		maxRetries int32
		expected   bool
	}{
		{TaskStatusPending, 0, 3, false},
		{TaskStatusRunning, 0, 3, false},
		{TaskStatusSucceeded, 0, 3, false},
		{TaskStatusFailed, 0, 3, true},
		{TaskStatusFailed, 1, 3, true},
		{TaskStatusFailed, 2, 3, true},
		{TaskStatusFailed, 3, 3, false}, // 达到最大重试次数
		{TaskStatusFailed, 4, 3, false}, // 超过最大重试次数
		{TaskStatusCancelled, 0, 3, false},
		{TaskStatusTimeout, 0, 3, false},
	}

	for _, tt := range tests {
		task := &Task{
			Status:     tt.status,
			RetryCount: tt.retryCount,
			MaxRetries: tt.maxRetries,
		}
		result := task.CanRetry()
		if result != tt.expected {
			t.Errorf("Task.CanRetry() with Status=%s, RetryCount=%d, MaxRetries=%d = %v, expected %v",
				tt.status, tt.retryCount, tt.maxRetries, result, tt.expected)
		}
	}
}

func TestTask_MarkRunning(t *testing.T) {
	task := &Task{Name: "Test Task"}

	task.MarkRunning()

	if task.Status != TaskStatusRunning {
		t.Errorf("expected Status RUNNING, got %v", task.Status)
	}
	if task.StartedAt == nil {
		t.Error("StartedAt should be set")
	}
}

func TestTask_MarkCompleted(t *testing.T) {
	task := &Task{Name: "Test Task"}

	task.MarkCompleted()

	if task.Status != TaskStatusSucceeded {
		t.Errorf("expected Status SUCCEEDED, got %v", task.Status)
	}
	if task.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestTask_MarkFailed(t *testing.T) {
	task := &Task{Name: "Test Task", RetryCount: 0, MaxRetries: 3}

	errMsg := "test error message"
	task.MarkFailed(errMsg)

	if task.Status != TaskStatusFailed {
		t.Errorf("expected Status FAILED, got %v", task.Status)
	}
	if task.ErrorMessage != errMsg {
		t.Errorf("expected ErrorMessage '%s', got '%s'", errMsg, task.ErrorMessage)
	}
	if task.RetryCount != 1 {
		t.Errorf("expected RetryCount 1, got %d", task.RetryCount)
	}
}

func TestTaskEvent(t *testing.T) {
	now := time.Now()
	event := TaskEvent{
		ID:         "event-1",
		TaskID:     "task-1",
		FromStatus: TaskStatusPending,
		ToStatus:   TaskStatusRunning,
		Message:    "task started",
		Timestamp:  now,
		Operator:   "system",
	}

	if event.ID != "event-1" {
		t.Errorf("expected ID 'event-1', got '%s'", event.ID)
	}
	if event.TaskID != "task-1" {
		t.Errorf("expected TaskID 'task-1', got '%s'", event.TaskID)
	}
	if event.FromStatus != TaskStatusPending {
		t.Errorf("expected FromStatus PENDING, got %v", event.FromStatus)
	}
	if event.ToStatus != TaskStatusRunning {
		t.Errorf("expected ToStatus RUNNING, got %v", event.ToStatus)
	}
	if event.Message != "task started" {
		t.Errorf("expected Message 'task started', got '%s'", event.Message)
	}
	if event.Operator != "system" {
		t.Errorf("expected Operator 'system', got '%s'", event.Operator)
	}
}
