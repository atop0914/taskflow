package service

import (
	"fmt"
	"taskflow/internal/model"
	"time"
)

// StateMachine 任务状态机
type StateMachine struct {
	// transitions 定义有效状态转换
	transitions map[model.TaskStatus][]model.TaskStatus
}

// NewStateMachine 创建状态机
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		transitions: make(map[model.TaskStatus][]model.TaskStatus),
	}
	sm.initTransitions()
	return sm
}

// initTransitions 初始化有效状态转换
func (sm *StateMachine) initTransitions() {
	// PENDING 可以转换到 RUNNING, CANCELLED
	sm.transitions[model.TaskStatusPending] = []model.TaskStatus{
		model.TaskStatusRunning,
		model.TaskStatusCancelled,
	}

	// RUNNING 可以转换到 SUCCEEDED, FAILED, TIMEOUT, CANCELLED
	sm.transitions[model.TaskStatusRunning] = []model.TaskStatus{
		model.TaskStatusSucceeded,
		model.TaskStatusFailed,
		model.TaskStatusTimeout,
		model.TaskStatusCancelled,
	}

	// FAILED 可以转换到 PENDING (重试), CANCELLED
	sm.transitions[model.TaskStatusFailed] = []model.TaskStatus{
		model.TaskStatusPending,
		model.TaskStatusCancelled,
	}

	// 终态: SUCCEEDED, CANCELLED, TIMEOUT 不能转换到其他状态
	sm.transitions[model.TaskStatusSucceeded] = []model.TaskStatus{}
	sm.transitions[model.TaskStatusCancelled] = []model.TaskStatus{}
	sm.transitions[model.TaskStatusTimeout] = []model.TaskStatus{}

	// UNSPECIFIED 是初始态，可以转到 PENDING
	sm.transitions[model.TaskStatusUnspecified] = []model.TaskStatus{
		model.TaskStatusPending,
	}
}

// CanTransition 检查状态转换是否有效
func (sm *StateMachine) CanTransition(from, to model.TaskStatus) bool {
	allowed, exists := sm.transitions[from]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}

// Transition 执行状态转换
func (sm *StateMachine) Transition(task *model.Task, toStatus model.TaskStatus, operator string) error {
	fromStatus := task.Status

	// 验证转换
	if !sm.CanTransition(fromStatus, toStatus) {
		return fmt.Errorf("invalid state transition from %s to %s", fromStatus, toStatus)
	}

	// 执行转换前的钩子
	if err := sm.preTransition(task, fromStatus, toStatus); err != nil {
		return err
	}

	// 更新任务状态
	task.Status = toStatus

	// 执行转换后的钩子
	sm.postTransition(task, fromStatus, toStatus, operator)

	return nil
}

// preTransition 转换前钩子
func (sm *StateMachine) preTransition(task *model.Task, from, to model.TaskStatus) error {
	// 可以在这里添加业务逻辑验证
	switch to {
	case model.TaskStatusRunning:
		if task.StartedAt == nil {
			// 将在 postTransition 中设置
		}
	case model.TaskStatusSucceeded:
		if task.CompletedAt == nil {
			// 将在 postTransition 中设置
		}
	}
	return nil
}

// postTransition 转换后钩子
func (sm *StateMachine) postTransition(task *model.Task, from, to model.TaskStatus, operator string) {
	now := time.Now()

	switch to {
	case model.TaskStatusRunning:
		task.StartedAt = &now
	case model.TaskStatusSucceeded:
		task.CompletedAt = &now
		task.ErrorMessage = ""
	case model.TaskStatusFailed:
		task.RetryCount++
	case model.TaskStatusCancelled:
		// 取消时记录时间
		if task.StartedAt != nil && task.CompletedAt == nil {
			task.CompletedAt = &now
		}
	case model.TaskStatusTimeout:
		if task.StartedAt != nil && task.CompletedAt == nil {
			task.CompletedAt = &now
		}
	}

	task.UpdatedAt = now
}

// GetAllowedTransitions 获取允许的状态转换列表
func (sm *StateMachine) GetAllowedTransitions(status model.TaskStatus) []model.TaskStatus {
	return sm.transitions[status]
}

// IsTerminal 检查是否为终态
func (sm *StateMachine) IsTerminal(status model.TaskStatus) bool {
	return status == model.TaskStatusSucceeded ||
		status == model.TaskStatusFailed ||
		status == model.TaskStatusCancelled ||
		status == model.TaskStatusTimeout
}
