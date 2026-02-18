package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"taskflow/internal/model"
	"taskflow/internal/repository"
)

// Scheduler 任务调度器
type Scheduler struct {
	repo            *repository.TaskRepository
	stateMachine    *StateMachine
	depChecker      *DefaultDependencyChecker
	workerPool      *WorkerPool
	pollingInterval time.Duration
	maxPending      int

	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	// 状态
	statusMu     sync.RWMutex
	pendingCnt   int
	runningCnt   int
	scheduledCnt int
	finishedCnt  int
}

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
	IsRunning   bool   `json:"is_running"`
	PendingCnt  int    `json:"pending_count"`
	RunningCnt  int    `json:"running_count"`
	ScheduledCnt int   `json:"scheduled_count"`
	FinishedCnt int    `json:"finished_count"`
	WorkerCount int    `json:"worker_count"`
}

// WorkerPool 工作池
type WorkerPool struct {
	size    int
	workers chan struct{}
	tasks   chan string // task IDs
	wg      sync.WaitGroup
}

// NewWorkerPool 创建工作池
func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{
		size:    size,
		workers: make(chan struct{}, size),
		tasks:   make(chan string, size*2),
	}
}

// Run 开始处理任务
func (wp *WorkerPool) Run(handler func(taskID string)) {
	for i := 0; i < wp.size; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for taskID := range wp.tasks {
				handler(taskID)
			}
		}()
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(taskID string) bool {
	select {
	case wp.tasks <- taskID:
		return true
	default:
		return false
	}
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	close(wp.tasks)
	wp.wg.Wait()
}

// NewScheduler 创建调度器
func NewScheduler(repo *repository.TaskRepository) *Scheduler {
	s := &Scheduler{
		repo:            repo,
		stateMachine:    NewStateMachine(),
		depChecker:      NewDefaultDependencyChecker(repo),
		pollingInterval: 5 * time.Second,
		maxPending:      100,
	}

	// 默认 10 个 worker
	s.workerPool = NewWorkerPool(10)

	// 设置任务处理函数
	s.setupTaskHandler()

	return s
}

// setupTaskHandler 设置任务处理函数
func (s *Scheduler) setupTaskHandler() {
	s.workerPool.Run(func(taskID string) {
		s.executeTask(taskID)
	})
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	// 启动轮询循环
	go s.pollingLoop()

	log.Printf("Scheduler started")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.running = false
	s.workerPool.Stop()

	log.Printf("Scheduler stopped")
}

// GetStatus 获取调度器状态
func (s *Scheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()

	return SchedulerStatus{
		IsRunning:   s.running,
		PendingCnt:  s.pendingCnt,
		RunningCnt:  s.runningCnt,
		ScheduledCnt: s.scheduledCnt,
		FinishedCnt: s.finishedCnt,
		WorkerCount: s.workerPool.size,
	}
}

// pollingLoop 轮询待处理任务
func (s *Scheduler) pollingLoop() {
	ticker := time.NewTicker(s.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.pollPendingTasks()
		}
	}
}

// pollPendingTasks 轮询并调度待处理任务
func (s *Scheduler) pollPendingTasks() {
	tasks, err := s.repo.ListPending(s.maxPending)
	if err != nil {
		log.Printf("Failed to list pending tasks: %v", err)
		return
	}

	for _, task := range tasks {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.TrySchedule(task.ID)
		}
	}

	s.statusMu.Lock()
	s.pendingCnt = len(tasks)
	s.statusMu.Unlock()
}

// TrySchedule 尝试调度任务
func (s *Scheduler) TrySchedule(taskID string) error {
	s.statusMu.RLock()
	running := s.running
	s.statusMu.RUnlock()

	if !running {
		return nil
	}

	// 检查依赖
	ready, err := s.depChecker.CheckDependencies(taskID)
	if err != nil {
		log.Printf("Failed to check dependencies for task %s: %v", taskID, err)
		return err
	}
	if !ready {
		return nil // 依赖未满足，等待
	}

	// 获取任务
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	// 检查任务状态
	if task.Status != model.TaskStatusPending {
		return nil
	}

	// 原子更新状态为 RUNNING
	err = s.repo.UpdateStatusWithEvent(taskID, model.TaskStatusPending, model.TaskStatusRunning, "scheduler", "task scheduled")
	if err != nil {
		log.Printf("Failed to schedule task %s: %v", taskID, err)
		return err
	}

	// 提交到工作池
	if s.workerPool.Submit(taskID) {
		s.statusMu.Lock()
		s.scheduledCnt++
		s.statusMu.Unlock()
		log.Printf("Task %s scheduled", taskID)
	}

	return nil
}

// executeTask 执行任务
func (s *Scheduler) executeTask(taskID string) {
	s.statusMu.Lock()
	s.runningCnt++
	s.statusMu.Unlock()

	defer func() {
		s.statusMu.Lock()
		s.runningCnt--
		s.statusMu.Unlock()
	}()

	log.Printf("Executing task %s", taskID)

	// 获取最新任务状态
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		return
	}

	// 检查是否被取消
	if task.Status == model.TaskStatusCancelled {
		log.Printf("Task %s was cancelled", taskID)
		return
	}

	// 执行业务逻辑（这里应该是可扩展的 handler）
	result, err := s.executeTaskHandler(task)
	if err != nil {
		// 执行失败，更新状态
		s.handleTaskFailure(taskID, err.Error())
		return
	}

	// 执行成功
	s.handleTaskSuccess(taskID, result)
}

// executeTaskHandler 实际执行任务逻辑
func (s *Scheduler) executeTaskHandler(task *model.Task) (map[string]string, error) {
	// TODO: 实现具体的任务执行逻辑
	// 这里可以扩展为根据 task.TaskType 调用不同的处理器

	log.Printf("Running task %s of type %s", task.ID, task.TaskType)

	// 模拟执行
	time.Sleep(100 * time.Millisecond)

	// 返回结果
	return map[string]string{
		"status": "completed",
		"output": "task executed successfully",
	}, nil
}

// handleTaskSuccess 处理任务成功
func (s *Scheduler) handleTaskSuccess(taskID string, result map[string]string) {
	err := s.repo.UpdateStatusWithEvent(taskID, model.TaskStatusRunning, model.TaskStatusSucceeded, "scheduler", "task completed")
	if err != nil {
		log.Printf("Failed to update task %s status: %v", taskID, err)
		return
	}

	// 更新任务输出结果
	task, err := s.repo.GetByID(taskID)
	if err == nil && task != nil {
		task.OutputResult = result
		s.repo.Update(task)
	}

	s.statusMu.Lock()
	s.finishedCnt++
	s.statusMu.Unlock()

	log.Printf("Task %s succeeded", taskID)

	// 检查依赖此任务的其他任务
	s.checkDependentTasks(taskID)
}

// handleTaskFailure 处理任务失败
func (s *Scheduler) handleTaskFailure(taskID string, errMsg string) {
	task, err := s.repo.GetByID(taskID)
	if err != nil || task == nil {
		return
	}

	// 检查是否可以重试
	if task.CanRetry() {
		// 重置为 Pending，等待下次调度
		err = s.repo.UpdateStatusWithEvent(taskID, model.TaskStatusRunning, model.TaskStatusPending, "scheduler", fmt.Sprintf("retry: %s", errMsg))
		log.Printf("Task %s failed, will retry (attempt %d/%d)", taskID, task.RetryCount+1, task.MaxRetries)
	} else {
		// 标记为失败
		err = s.repo.UpdateStatusWithEvent(taskID, model.TaskStatusRunning, model.TaskStatusFailed, "scheduler", errMsg)
		log.Printf("Task %s failed permanently", taskID)
	}

	if err != nil {
		log.Printf("Failed to update task %s status: %v", taskID, err)
	}
}

// checkDependentTasks 检查依赖此任务的其他任务
func (s *Scheduler) checkDependentTasks(completedTaskID string) {
	// TODO: 实现依赖查询
	// 目前需要通过其他方式触发下游任务调度
	log.Printf("Checking dependent tasks for %s", completedTaskID)
}

// SetWorkerCount 设置 worker 数量
func (s *Scheduler) SetWorkerCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止旧的 worker pool
	if s.running {
		s.workerPool.Stop()
		s.workerPool = NewWorkerPool(count)
		s.setupTaskHandler()
	}
}

// SetPollingInterval 设置轮询间隔
func (s *Scheduler) SetPollingInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollingInterval = interval
}
