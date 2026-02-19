package service

import (
	"context"
	"os"
	"sync"
	"testing"

	"taskflow/internal/model"
	"taskflow/internal/repository"
)

func setupTestService(t *testing.T) (*TaskService, *repository.TaskRepository, func()) {
	tmpFile, err := os.CreateTemp("", "taskflow_service_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	db, err := repository.NewSQLite(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create SQLite: %v", err)
	}

	if err := db.InitSchema(); err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to init schema: %v", err)
	}

	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo)

	cleanup := func() {
		service.StopScheduler()
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return service, repo, cleanup
}

func TestTaskService_CreateTask(t *testing.T) {
	service, repo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	task, err := service.CreateTask(
		ctx,
		"Test Task",
		"Test Description",
		model.TaskPriorityNormal,
		"test",
		map[string]string{"key": "value"},
		nil,
		3,
		"testuser",
	)

	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if task == nil {
		t.Fatal("task should not be nil")
	}
	if task.Name != "Test Task" {
		t.Errorf("expected name 'Test Task', got '%s'", task.Name)
	}
	if task.Status != model.TaskStatusPending {
		t.Errorf("expected status PENDING, got %v", task.Status)
	}

	// 验证数据库中创建成功
	dbTask, err := repo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("failed to get task from db: %v", err)
	}
	if dbTask == nil {
		t.Error("task should exist in database")
	}
}

func TestTaskService_CreateTaskWithDependencies(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 先创建一个依赖任务
	depTask, err := service.CreateTask(
		ctx,
		"Dependency Task",
		"First",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create dependency task: %v", err)
	}

	// 创建依赖任务
	task, err := service.CreateTask(
		ctx,
		"Dependent Task",
		"Second",
		model.TaskPriorityNormal,
		"test",
		nil,
		[]string{depTask.ID},
		3,
		"testuser",
	)

	if err != nil {
		t.Fatalf("failed to create task with dependencies: %v", err)
	}
	if len(task.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(task.Dependencies))
	}
	if task.Dependencies[0] != depTask.ID {
		t.Errorf("expected dependency '%s', got '%s'", depTask.ID, task.Dependencies[0])
	}
}

func TestTaskService_GetTask(t *testing.T) {
	service, repo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建任务
	created, err := service.CreateTask(
		ctx,
		"Get Test",
		"desc",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 获取任务
	task, err := service.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task == nil {
		t.Fatal("task should not be nil")
	}
	if task.ID != created.ID {
		t.Errorf("expected ID '%s', got '%s'", created.ID, task.ID)
	}

	// 获取不存在的任务
	notFound, err := service.GetTask(ctx, "non-existent")
	if err != nil {
		t.Fatalf("error should be nil for not found: %v", err)
	}
	if notFound != nil {
		t.Error("task should be nil for non-existent ID")
	}

	_ = repo // silence unused warning
}

func TestTaskService_UpdateTask(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建任务
	task, err := service.CreateTask(
		ctx,
		"Update Test",
		"original",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 更新任务
	updates := map[string]interface{}{
		"status":       model.TaskStatusRunning,
		"output_result": map[string]string{"result": "success"},
	}
	updated, err := service.UpdateTask(ctx, task.ID, updates, "test-operator")
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}
	if updated.Status != model.TaskStatusRunning {
		t.Errorf("expected status RUNNING, got %v", updated.Status)
	}
}

func TestTaskService_CancelTask(t *testing.T) {
	service, repo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建任务
	task, err := service.CreateTask(
		ctx,
		"Cancel Test",
		"desc",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 先将任务更新为 Running 状态
	task.Status = model.TaskStatusRunning
	if err := repo.Update(task); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	// 取消任务
	err = service.CancelTask(ctx, task.ID, "test-operator")
	if err != nil {
		t.Fatalf("failed to cancel task: %v", err)
	}

	// 验证状态
	updated, err := service.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if updated.Status != model.TaskStatusCancelled {
		t.Errorf("expected status CANCELLED, got %v", updated.Status)
	}

	// 测试取消已终态的任务
	err = service.CancelTask(ctx, task.ID, "test-operator")
	if err == nil {
		t.Error("expected error when cancelling terminal task")
	}
}

func TestTaskService_RetryTask(t *testing.T) {
	service, repo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建任务
	task, err := service.CreateTask(
		ctx,
		"Retry Test",
		"desc",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 先将任务标记为失败
	task.Status = model.TaskStatusFailed
	task.RetryCount = 0
	task.MaxRetries = 3
	if err := repo.Update(task); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	// 重试任务
	err = service.RetryTask(ctx, task.ID, "test-operator")
	if err != nil {
		t.Fatalf("failed to retry task: %v", err)
	}

	// 验证状态
	updated, err := service.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if updated.Status != model.TaskStatusPending {
		t.Errorf("expected status PENDING after retry, got %v", updated.Status)
	}
}

func TestTaskService_Scheduler(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建任务
	task, err := service.CreateTask(
		ctx,
		"Scheduler Test",
		"desc",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 启动调度器
	service.StartScheduler(ctx)

	// 等待任务被调度
	// 注意：调度器异步执行，需要等待
	status := service.GetSchedulerStatus()
	if !status.IsRunning {
		t.Error("scheduler should be running")
	}

	// 停止调度器
	service.StopScheduler()

	status = service.GetSchedulerStatus()
	if status.IsRunning {
		t.Error("scheduler should be stopped")
	}

	_ = task // silence unused warning
}

func TestTaskService_ListTasks(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建多个任务
	for i := 0; i < 5; i++ {
		_, err := service.CreateTask(
			ctx,
			"List Test Task",
			"desc",
			model.TaskPriorityNormal,
			"test",
			nil,
			nil,
			3,
			"testuser",
		)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// 列出任务
	filter := repository.TaskFilter{PageSize: 10, PageIndex: 0}
	tasks, total, err := service.ListTasks(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if total != 5 {
		t.Errorf("expected 5 total tasks, got %d", total)
	}
	if len(tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tasks))
	}
}

func TestTaskService_SearchTasks(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建任务
	_, err := service.CreateTask(
		ctx,
		"Unique Search Key Task",
		"description",
		model.TaskPriorityNormal,
		"test",
		nil,
		nil,
		3,
		"testuser",
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 搜索
	tasks, err := service.SearchTasks(ctx, "Unique", 10, 0)
	if err != nil {
		t.Fatalf("failed to search tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestDependencyChecker_CheckDependencies(t *testing.T) {
	_, repo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建依赖任务
	depTask := model.NewTask("Dep Task", "desc", model.TaskPriorityNormal, "test", nil, nil, 3, "test")
	depTask.ID = "dep-1"
	depTask.Status = model.TaskStatusSucceeded
	if err := repo.Create(depTask); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 创建被依赖的任务
	mainTask := model.NewTask("Main Task", "desc", model.TaskPriorityNormal, "test", nil, []string{"dep-1"}, 3, "test")
	mainTask.ID = "main-1"
	mainTask.Status = model.TaskStatusPending
	if err := repo.Create(mainTask); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	checker := NewDefaultDependencyChecker(repo)

	// 测试依赖满足
	ready, err := checker.CheckDependencies("main-1")
	if err != nil {
		t.Fatalf("failed to check dependencies: %v", err)
	}
	if !ready {
		t.Error("dependencies should be satisfied")
	}

	// 测试依赖不满足
	depTask.Status = model.TaskStatusRunning
	repo.Update(depTask)

	ready, err = checker.CheckDependencies("main-1")
	if err != nil {
		t.Fatalf("failed to check dependencies: %v", err)
	}
	if ready {
		t.Error("dependencies should not be satisfied")
	}

	_ = ctx // silence unused warning
}

func TestWorkerPool(t *testing.T) {
	pool := NewWorkerPool(2)
	
	executed := make(chan string, 10)
	var wg sync.WaitGroup
	wg.Add(2)
	
	// Start workers
	pool.Run(func(taskID string) {
		executed <- taskID
		wg.Done()
	})
	
	// Submit tasks
	pool.Submit("task-1")
	pool.Submit("task-2")
	
	// Wait for completion
	wg.Wait()
	close(executed)
	pool.Stop()
	
	results := make([]string, 0)
	for taskID := range executed {
		results = append(results, taskID)
	}
	
	if len(results) != 2 {
		t.Errorf("expected 2 executed tasks, got %d", len(results))
	}
}
