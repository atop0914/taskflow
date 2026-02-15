package handler

import "log"

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	errorcode "taskflow/internal/error"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	pb "taskflow/proto"
)

// TaskHandler 任务处理器
type TaskHandler struct {
	repo *repository.TaskRepository
	pb.UnimplementedTaskServiceServer
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(repo *repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.Task, error) {
	// 参数验证
	if req.Name == "" {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeInvalidParam, "name is required").ToGRPCStatus().Err()
	}

	// 创建任务模型
	task := model.NewTask(
		req.Name,
		req.Description,
		model.TaskPriority(req.Priority),
		req.TaskType,
		req.InputParams,
		req.Dependencies,
		req.MaxRetries,
		req.CreatedBy,
	)
	task.ID = uuid.New().String()

	// 保存到数据库
	if err := h.repo.Create(task); err != nil {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeDBError, err.Error()).ToGRPCStatus().Err()
	}

	return h.toPBTask(task, false), nil
}

// GetTask 获取任务
func (h *TaskHandler) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	if req.Id == "" {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeInvalidParam, "id is required").ToGRPCStatus().Err()
	}

	task, err := h.repo.GetByID(req.Id)
	if err != nil { log.Printf("Handler error: %v", err)
		return nil, errorcode.NewTaskError(errorcode.ErrCodeDBError, err.Error()).ToGRPCStatus().Err()
	}
	if task == nil {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeTaskNotFound, "task not found").ToGRPCStatus().Err()
	}

	return h.toPBTask(task, req.IncludeEvents), nil
}

// ListTasks 列出任务
func (h *TaskHandler) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	// 分页参数
	pageSize := int(req.PageSize)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	pageIndex := int(req.Page)
	if pageIndex < 0 {
		pageIndex = 0
	}
	offset := pageIndex * pageSize

	// 构建过滤条件
	filter := repository.TaskFilter{
		PageSize:  pageSize,
		PageIndex: offset,
		Keyword:   req.Keyword,
		TaskType:  req.TaskType,
	}

	if len(req.StatusFilter) > 0 {
		status := model.TaskStatus(req.StatusFilter[0])
		filter.Status = &status
	}
	if req.Priority != 0 {
		priority := model.TaskPriority(req.Priority)
		filter.Priority = &priority
	}

	// 查询
	tasks, total, err := h.repo.ListByFilter(filter)
	if err != nil { log.Printf("Handler error: %v", err)
		return nil, errorcode.NewTaskError(errorcode.ErrCodeDBError, err.Error()).ToGRPCStatus().Err()
	}

	// 转换
	pbTasks := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		pbTasks[i] = h.toPBTask(task, false)
	}

	return &pb.ListTasksResponse{
		Tasks:    pbTasks,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// UpdateTask 更新任务
func (h *TaskHandler) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.Task, error) {
	if req.Id == "" {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeInvalidParam, "id is required").ToGRPCStatus().Err()
	}

	// 获取现有任务
	task, err := h.repo.GetByID(req.Id)
	if err != nil { log.Printf("Handler error: %v", err)
		return nil, errorcode.NewTaskError(errorcode.ErrCodeDBError, err.Error()).ToGRPCStatus().Err()
	}
	if task == nil {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeTaskNotFound, "task not found").ToGRPCStatus().Err()
	}

	// 更新字段
	if req.Status != 0 {
		oldStatus := task.Status
		newStatus := model.TaskStatus(req.Status)

		// 状态转换验证
		if !isValidStatusTransition(oldStatus, newStatus) {
			return nil, errorcode.NewTaskError(errorcode.ErrCodeInvalidState,
				fmt.Sprintf("invalid status transition from %s to %s", oldStatus, newStatus)).ToGRPCStatus().Err()
		}

		// 原子更新状态
		err := h.repo.UpdateStatusWithEvent(req.Id, oldStatus, newStatus, "system", "status updated")
		if err != nil { log.Printf("Handler error: %v", err)
			return nil, errorcode.NewTaskError(errorcode.ErrCodeDBError, err.Error()).ToGRPCStatus().Err()
		}
		task.Status = newStatus
	}

	if req.OutputResult != nil {
		task.OutputResult = req.OutputResult
	}
	if req.ErrorMessage != "" {
		task.ErrorMessage = req.ErrorMessage
	}
	if req.RetryCount != 0 {
		task.RetryCount = req.RetryCount
	}
	task.UpdatedAt = time.Now()

	// 保存
	if err := h.repo.Update(task); err != nil {
		return nil, errorcode.NewTaskError(errorcode.ErrCodeDBError, err.Error()).ToGRPCStatus().Err()
	}

	return h.toPBTask(task, false), nil
}

// 状态转换验证
func isValidStatusTransition(from, to model.TaskStatus) bool {
	// PENDING 可以转到 RUNNING, CANCELLED
	if from == model.TaskStatusPending {
		return to == model.TaskStatusRunning || to == model.TaskStatusCancelled
	}
	// RUNNING 可以转到 SUCCEEDED, FAILED, TIMEOUT, CANCELLED
	if from == model.TaskStatusRunning {
		return to == model.TaskStatusSucceeded ||
			to == model.TaskStatusFailed ||
			to == model.TaskStatusTimeout ||
			to == model.TaskStatusCancelled
	}
	// 终态不能转换
	return false
}

// toPBTask 转换为 Protobuf 任务
func (h *TaskHandler) toPBTask(task *model.Task, includeEvents bool) *pb.Task {
	pbTask := &pb.Task{
		Id:           task.ID,
		Name:         task.Name,
		Description:  task.Description,
		Status:       pb.TaskStatus(task.Status),
		Priority:     pb.TaskPriority(task.Priority),
		TaskType:     task.TaskType,
		InputParams:  task.InputParams,
		OutputResult: task.OutputResult,
		Dependencies: task.Dependencies,
		RetryCount:   task.RetryCount,
		MaxRetries:   task.MaxRetries,
		ErrorMessage: task.ErrorMessage,
		CreatedAt:    task.CreatedAt.Unix(),
		UpdatedAt:    task.UpdatedAt.Unix(),
		CreatedBy:    task.CreatedBy,
	}

	if task.StartedAt != nil {
		pbTask.StartedAt = task.StartedAt.Unix()
	}
	if task.CompletedAt != nil {
		pbTask.CompletedAt = task.CompletedAt.Unix()
	}

	if includeEvents {
		for _, e := range task.Events {
			pbTask.Events = append(pbTask.Events, &pb.TaskEvent{
				Id:         e.ID,
				FromStatus: pb.TaskStatus(e.FromStatus),
				ToStatus:   pb.TaskStatus(e.ToStatus),
				Message:    e.Message,
				Timestamp:  e.Timestamp.Unix(),
				Operator:   e.Operator,
			})
		}
	}

	return pbTask
}

// RegisterTaskHandlers 注册任务服务句柄
func RegisterTaskHandlers(repo *repository.TaskRepository) *TaskHandler {
	return NewTaskHandler(repo)
}
