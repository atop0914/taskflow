package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"taskflow/internal/model"
	"taskflow/internal/repository"
)

// TaskService 任务服务
type TaskService struct {
	repo      *repository.TaskRepository
	scheduler *Scheduler
}

// NewTaskService 创建任务服务
func NewTaskService(repo *repository.TaskRepository) *TaskService {
	return &TaskService{
		repo:      repo,
		scheduler: NewScheduler(repo),
	}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, name, description string, priority model.TaskPriority, taskType string, inputParams map[string]string, dependencies []string, maxRetries int32, createdBy string) (*model.Task, error) {
	// 验证依赖任务是否存在
	for _, depID := range dependencies {
		depTask, err := s.repo.GetByID(depID)
		if err != nil {
			return nil, fmt.Errorf("failed to get dependency task: %w", err)
		}
		if depTask == nil {
			return nil, fmt.Errorf("dependency task not found: %s", depID)
		}
	}

	task := model.NewTask(name, description, priority, taskType, inputParams, dependencies, maxRetries, createdBy)
	task.ID = uuid.New().String()

	if err := s.repo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 记录创建事件
	s.recordEvent(task, model.TaskStatusUnspecified, model.TaskStatusPending, "task created", createdBy)

	// 检查是否可以调度
	if len(dependencies) == 0 {
		s.scheduler.TrySchedule(task.ID)
	}

	return task, nil
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return s.repo.GetByID(id)
}

// UpdateTask 更新任务
func (s *TaskService) UpdateTask(ctx context.Context, id string, updates map[string]interface{}, operator string) (*model.Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	// 应用更新
	if status, ok := updates["status"].(model.TaskStatus); ok {
		if err := s.scheduler.stateMachine.Transition(task, status, operator); err != nil {
			return nil, err
		}
		task.Status = status

		// 检查依赖任务的完成状态
		if status == model.TaskStatusSucceeded {
			s.checkAndScheduleDependencies(task)
		}
	}

	if result, ok := updates["output_result"].(map[string]string); ok {
		task.OutputResult = result
	}

	if errMsg, ok := updates["error_message"].(string); ok {
		task.ErrorMessage = errMsg
	}

	task.UpdatedAt = time.Now()

	if err := s.repo.Update(task); err != nil {
		return nil, err
	}

	return task, nil
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, id, operator string) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", id)
	}

	if task.IsTerminal() {
		return fmt.Errorf("cannot cancel terminal task")
	}

	fromStatus := task.Status
	if err := s.scheduler.stateMachine.Transition(task, model.TaskStatusCancelled, operator); err != nil {
		return err
	}

	// 保存到数据库
	return s.repo.UpdateStatusWithEvent(id, fromStatus, model.TaskStatusCancelled, operator, "task cancelled")
}

// RetryTask 重试任务
func (s *TaskService) RetryTask(ctx context.Context, id, operator string) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", id)
	}

	if !task.CanRetry() {
		return fmt.Errorf("task cannot be retried")
	}

	// 重置为 Pending 状态
	fromStatus := task.Status
	retryMsg := fmt.Sprintf("retry attempt %d", task.RetryCount+1)
	if err := s.scheduler.stateMachine.Transition(task, model.TaskStatusPending, retryMsg); err != nil {
		return err
	}

	// 保存到数据库
	return s.repo.UpdateStatusWithEvent(id, fromStatus, model.TaskStatusPending, operator, retryMsg)
}

// StartScheduler 启动调度器
func (s *TaskService) StartScheduler(ctx context.Context) {
	s.scheduler.Start(ctx)
}

// StopScheduler 停止调度器
func (s *TaskService) StopScheduler() {
	s.scheduler.Stop()
}

// GetSchedulerStatus 获取调度器状态
func (s *TaskService) GetSchedulerStatus() SchedulerStatus {
	return s.scheduler.GetStatus()
}

// recordEvent 记录任务事件
func (s *TaskService) recordEvent(task *model.Task, fromStatus, toStatus model.TaskStatus, message, operator string) {
	event := &model.TaskEvent{
		ID:         fmt.Sprintf("%s_%d", task.ID, time.Now().UnixNano()),
		TaskID:     task.ID,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		Message:    message,
		Timestamp:  time.Now(),
		Operator:   operator,
	}
	if err := s.repo.AddEvent(event); err != nil {
		log.Printf("failed to record event: %v", err)
	}
}

// checkAndScheduleDependencies 检查并调度依赖任务
func (s *TaskService) checkAndScheduleDependencies(completedTask *model.Task) {
	// 查找所有依赖此任务的任务
	// 这里需要实现依赖查询逻辑，暂时简化处理
	log.Printf("Task %s completed, checking dependencies", completedTask.ID)
}

// ListTasks 列出任务
func (s *TaskService) ListTasks(ctx context.Context, filter repository.TaskFilter) ([]*model.Task, int, error) {
	return s.repo.ListByFilter(filter)
}

// SearchTasks 搜索任务
func (s *TaskService) SearchTasks(ctx context.Context, keyword string, limit, offset int) ([]*model.Task, error) {
	return s.repo.Search(keyword, limit, offset)
}

// GetTaskEvents 获取任务事件
func (s *TaskService) GetTaskEvents(ctx context.Context, taskID string) ([]model.TaskEvent, error) {
	return s.repo.GetEventsByTaskID(taskID)
}

// DependencyChecker 依赖检查器接口
type DependencyChecker interface {
	CheckDependencies(taskID string) (bool, error)
}

// DefaultDependencyChecker 默认依赖检查器
type DefaultDependencyChecker struct {
	repo *repository.TaskRepository
}

func NewDefaultDependencyChecker(repo *repository.TaskRepository) *DefaultDependencyChecker {
	return &DefaultDependencyChecker{repo: repo}
}

// CheckDependencies 检查任务的所有依赖是否都已完成
func (c *DefaultDependencyChecker) CheckDependencies(taskID string) (bool, error) {
	task, err := c.repo.GetByID(taskID)
	if err != nil {
		return false, err
	}
	if task == nil {
		return false, fmt.Errorf("task not found: %s", taskID)
	}

	// 没有依赖，直接可调度
	if len(task.Dependencies) == 0 {
		return true, nil
	}

	// 检查所有依赖任务是否都已完成
	for _, depID := range task.Dependencies {
		depTask, err := c.repo.GetByID(depID)
		if err != nil {
			return false, err
		}
		if depTask == nil {
			return false, fmt.Errorf("dependency task not found: %s", depID)
		}
		// 只有成功完成的任务才能触发下游
		if depTask.Status != model.TaskStatusSucceeded {
			return false, nil
		}
	}

	return true, nil
}
