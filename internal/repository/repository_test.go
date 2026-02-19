package repository

import (
	"os"
	"testing"

	"taskflow/internal/model"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) (*SQLite, func()) {
	tmpFile, err := os.CreateTemp("", "taskflow_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	db, err := NewSQLite(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create SQLite: %v", err)
	}

	if err := db.InitSchema(); err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to init schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestTaskRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	task := &model.Task{
		ID:          "test-1",
		Name:        "Test Task",
		Description: "A test task",
		Status:      model.TaskStatusPending,
		Priority:    model.TaskPriorityNormal,
		TaskType:    "test",
		InputParams: map[string]string{"key": "value"},
		MaxRetries:  3,
		CreatedBy:   "testuser",
	}

	err := repo.Create(task)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 验证创建成功
	task, err = repo.GetByID("test-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task == nil {
		t.Fatal("task should not be nil")
	}
	if task.Name != "Test Task" {
		t.Errorf("expected name 'Test Task', got '%s'", task.Name)
	}
}

func TestTaskRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建测试任务
	task := model.NewTask("Get Test", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task.ID = "get-test-1"
	if err := repo.Create(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 测试获取
	retrieved, err := repo.GetByID("get-test-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if retrieved == nil {
		t.Fatal("task should not be nil")
	}
	if retrieved.ID != "get-test-1" {
		t.Errorf("expected ID 'get-test-1', got '%s'", retrieved.ID)
	}

	// 测试不存在的任务
	notFound, err := repo.GetByID("non-existent")
	if err != nil {
		t.Fatalf("error should be nil for not found: %v", err)
	}
	if notFound != nil {
		t.Error("task should be nil for non-existent ID")
	}
}

func TestTaskRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	task := model.NewTask("Update Test", "original desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task.ID = "update-test-1"
	if err := repo.Create(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 更新任务
	task.Description = "updated desc"
	task.Status = model.TaskStatusRunning
	if err := repo.Update(task); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	// 验证更新
	updated, err := repo.GetByID("update-test-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if updated.Description != "updated desc" {
		t.Errorf("expected description 'updated desc', got '%s'", updated.Description)
	}
	if updated.Status != model.TaskStatusRunning {
		t.Errorf("expected status RUNNING, got %v", updated.Status)
	}
}

func TestTaskRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	task := model.NewTask("Delete Test", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task.ID = "delete-test-1"
	if err := repo.Create(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 删除任务
	if err := repo.Delete("delete-test-1"); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// 验证删除
	deleted, err := repo.GetByID("delete-test-1")
	if err != nil {
		t.Fatalf("error should be nil: %v", err)
	}
	if deleted != nil {
		t.Error("task should be nil after delete")
	}
}

func TestTaskRepository_ListByStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建多个任务，不同状态
	tasks := []*model.Task{
		model.NewTask("Task 1", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test"),
		model.NewTask("Task 2", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test"),
		model.NewTask("Task 3", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test"),
	}
	tasks[0].ID = "list-test-1"
	tasks[0].Status = model.TaskStatusPending
	tasks[1].ID = "list-test-2"
	tasks[1].Status = model.TaskStatusRunning
	tasks[2].ID = "list-test-3"
	tasks[2].Status = model.TaskStatusSucceeded

	for _, task := range tasks {
		if err := repo.Create(task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// 测试列出 Pending 任务
	pending, err := repo.ListByStatus(model.TaskStatusPending, 10)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending task, got %d", len(pending))
	}

	// 测试列出 Running 任务
	running, err := repo.ListByStatus(model.TaskStatusRunning, 10)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(running) != 1 {
		t.Errorf("expected 1 running task, got %d", len(running))
	}

	// 测试列出 Succeeded 任务
	succeeded, err := repo.ListByStatus(model.TaskStatusSucceeded, 10)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(succeeded) != 1 {
		t.Errorf("expected 1 succeeded task, got %d", len(succeeded))
	}
}

func TestTaskRepository_ListPending(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建多个待处理任务
	for i := 0; i < 5; i++ {
		task := model.NewTask("Pending Task", "desc", model.TaskPriority(i%4), "test", nil, nil, 3, "test")
		task.ID = "pending-test-" + string(rune('1'+i))
		task.Status = model.TaskStatusPending
		if err := repo.Create(task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// 创建已完成任务
	doneTask := model.NewTask("Done Task", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	doneTask.ID = "done-test-1"
	doneTask.Status = model.TaskStatusSucceeded
	if err := repo.Create(doneTask); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 列出待处理任务
	pending, err := repo.ListPending(10)
	if err != nil {
		t.Fatalf("failed to list pending tasks: %v", err)
	}
	if len(pending) != 5 {
		t.Errorf("expected 5 pending tasks, got %d", len(pending))
	}
}

func TestTaskRepository_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	task1 := model.NewTask("Count 1", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task1.ID = "count-test-1"
	task1.Status = model.TaskStatusPending
	task2 := model.NewTask("Count 2", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task2.ID = "count-test-2"
	task2.Status = model.TaskStatusPending

	if err := repo.Create(task1); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if err := repo.Create(task2); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 统计所有任务
	total, err := repo.Count(nil)
	if err != nil {
		t.Fatalf("failed to count tasks: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 total tasks, got %d", total)
	}

	// 统计 Pending 任务
	pending := model.TaskStatusPending
	pendingCount, err := repo.Count(&pending)
	if err != nil {
		t.Fatalf("failed to count pending tasks: %v", err)
	}
	if pendingCount != 2 {
		t.Errorf("expected 2 pending tasks, got %d", pendingCount)
	}
}

func TestTaskRepository_UpdateStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	task := model.NewTask("Status Test", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task.ID = "status-test-1"
	task.Status = model.TaskStatusPending
	if err := repo.Create(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 更新状态
	err := repo.UpdateStatus("status-test-1", model.TaskStatusPending, model.TaskStatusRunning)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	// 验证状态更新
	updated, err := repo.GetByID("status-test-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if updated.Status != model.TaskStatusRunning {
		t.Errorf("expected status RUNNING, got %v", updated.Status)
	}

	// 测试状态不匹配时更新失败
	err = repo.UpdateStatus("status-test-1", model.TaskStatusPending, model.TaskStatusRunning)
	if err == nil {
		t.Error("expected error for status mismatch")
	}
}

func TestTaskRepository_UpdateStatusWithEvent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	task := model.NewTask("Event Test", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task.ID = "event-test-1"
	task.Status = model.TaskStatusPending
	if err := repo.Create(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 更新状态并记录事件
	err := repo.UpdateStatusWithEvent("event-test-1", model.TaskStatusPending, model.TaskStatusRunning, "test-operator", "starting task")
	if err != nil {
		t.Fatalf("failed to update status with event: %v", err)
	}

	// 验证状态更新
	updated, err := repo.GetByID("event-test-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if updated.Status != model.TaskStatusRunning {
		t.Errorf("expected status RUNNING, got %v", updated.Status)
	}

	// 验证事件记录
	events, err := repo.GetEventsByTaskID("event-test-1")
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Message != "starting task" {
		t.Errorf("expected message 'starting task', got '%s'", events[0].Message)
	}
	if events[0].Operator != "test-operator" {
		t.Errorf("expected operator 'test-operator', got '%s'", events[0].Operator)
	}
}

func TestTaskRepository_AddEvent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	task := model.NewTask("Add Event Test", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	task.ID = "addevent-test-1"
	if err := repo.Create(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 添加事件
	event := &model.TaskEvent{
		ID:         "event-1",
		TaskID:     "addevent-test-1",
		FromStatus: model.TaskStatusUnspecified,
		ToStatus:   model.TaskStatusPending,
		Message:    "task created",
		Operator:   "system",
	}
	err := repo.AddEvent(event)
	if err != nil {
		t.Fatalf("failed to add event: %v", err)
	}

	// 获取事件
	events, err := repo.GetEventsByTaskID("addevent-test-1")
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestTaskRepository_Search(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	tasks := []*model.Task{
		model.NewTask("Go Build Task", "Build the project", model.TaskPriorityNormal, "build", nil, nil, 3, "test"),
		model.NewTask("Go Test Task", "Run tests", model.TaskPriorityNormal, "test", nil, nil, 3, "test"),
		model.NewTask("Python Script", "Run python script", model.TaskPriorityNormal, "script", nil, nil, 3, "test"),
	}
	tasks[0].ID = "search-test-1"
	tasks[1].ID = "search-test-2"
	tasks[2].ID = "search-test-3"

	for _, task := range tasks {
		if err := repo.Create(task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// 搜索
	results, err := repo.Search("Go", 10, 0)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// 搜索不存在
	notFound, err := repo.Search("xyz123", 10, 0)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(notFound) != 0 {
		t.Errorf("expected 0 results, got %d", len(notFound))
	}
}

func TestTaskRepository_ListByFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)

	// 创建任务
	tasks := []*model.Task{
		model.NewTask("Task 1", "desc", model.TaskPriorityHigh, "build", nil, nil, 3, "user1"),
		model.NewTask("Task 2", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "user1"),
		model.NewTask("Task 3", "desc", model.TaskPriorityLow, "build", nil, nil, 3, "user2"),
	}
	tasks[0].ID = "filter-test-1"
	tasks[1].ID = "filter-test-2"
	tasks[2].ID = "filter-test-3"

	for _, task := range tasks {
		if err := repo.Create(task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// 按状态过滤
	filter := TaskFilter{PageSize: 10, PageIndex: 0}
	status := model.TaskStatusPending
	filter.Status = &status
	results, total, err := repo.ListByFilter(filter)
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}
	if total != 3 {
		t.Errorf("expected 3 total, got %d", total)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// 按优先级过滤
	priority := model.TaskPriorityHigh
	filter2 := TaskFilter{Priority: &priority, PageSize: 10, PageIndex: 0}
	results2, _, err := repo.ListByFilter(filter2)
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}
	if len(results2) != 1 {
		t.Errorf("expected 1 result, got %d", len(results2))
	}

	// 按创建者过滤
	filter3 := TaskFilter{CreatedBy: "user1", PageSize: 10, PageIndex: 0}
	results3, _, err := repo.ListByFilter(filter3)
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}
	if len(results3) != 2 {
		t.Errorf("expected 2 results, got %d", len(results3))
	}
}
