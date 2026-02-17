package proto

import (
	proto "github.com/golang/protobuf/proto"
)

// ========== 流式 RPC 消息类型 ==========

// WatchTask 请求 - 监听任务状态变化
type WatchTaskRequest struct {
	TaskIds        []string    `protobuf:"bytes,1,rep,name=task_ids,json=taskIds" json:"task_ids,omitempty"`
	StatusFilter   []TaskStatus `protobuf:"varint,2,rep,packed,name=status_filter,json=statusFilter,enum=taskflow.TaskStatus" json:"status_filter,omitempty"`
	IncludeInitial bool        `protobuf:"varint,3,opt,name=include_initial,json=includeInitial" json:"include_initial,omitempty"`
}

func (x *WatchTaskRequest) Reset()         { *x = WatchTaskRequest{} }
func (x *WatchTaskRequest) String() string { return proto.CompactTextString(x) }
func (*WatchTaskRequest) ProtoMessage()    {}

func (x *WatchTaskRequest) GetTaskIds() []string {
	if x != nil {
		return x.TaskIds
	}
	return nil
}

func (x *WatchTaskRequest) GetStatusFilter() []TaskStatus {
	if x != nil {
		return x.StatusFilter
	}
	return nil
}

func (x *WatchTaskRequest) GetIncludeInitial() bool {
	if x != nil {
		return x.IncludeInitial
	}
	return false
}

// TaskChangeEvent 任务变更事件
type TaskChangeEvent struct {
	TaskId     string    `protobuf:"bytes,1,opt,name=task_id,json=taskId" json:"task_id,omitempty"`
	Task       *Task    `protobuf:"bytes,2,opt,name=task" json:"task,omitempty"`
	FromStatus TaskStatus `protobuf:"varint,3,enum=taskflow.TaskStatus,name=from_status,json=fromStatus" json:"from_status,omitempty"`
	ToStatus   TaskStatus `protobuf:"varint,4,enum=taskflow.TaskStatus,name=to_status,json=toStatus" json:"to_status,omitempty"`
	ChangedAt  int64     `protobuf:"varint,5,opt,name=changed_at,json=changedAt" json:"changed_at,omitempty"`
	ChangeType string    `protobuf:"bytes,6,opt,name=change_type,json=changeType" json:"change_type,omitempty"`
}

func (x *TaskChangeEvent) Reset()         { *x = TaskChangeEvent{} }
func (x *TaskChangeEvent) String() string { return proto.CompactTextString(x) }
func (*TaskChangeEvent) ProtoMessage()    {}

func (x *TaskChangeEvent) GetTaskId() string {
	if x != nil {
		return x.TaskId
	}
	return ""
}

func (x *TaskChangeEvent) GetTask() *Task {
	if x != nil {
		return x.Task
	}
	return nil
}

func (x *TaskChangeEvent) GetFromStatus() TaskStatus {
	if x != nil {
		return x.FromStatus
	}
	return TaskStatus_TASK_STATUS_UNSPECIFIED
}

func (x *TaskChangeEvent) GetToStatus() TaskStatus {
	if x != nil {
		return x.ToStatus
	}
	return TaskStatus_TASK_STATUS_UNSPECIFIED
}

func (x *TaskChangeEvent) GetChangedAt() int64 {
	if x != nil {
		return x.ChangedAt
	}
	return 0
}

func (x *TaskChangeEvent) GetChangeType() string {
	if x != nil {
		return x.ChangeType
	}
	return ""
}

// BatchCreateTasks 响应
type BatchCreateTasksResponse struct {
	Tasks        []*Task  `protobuf:"bytes,1,rep,name=tasks" json:"tasks,omitempty"`
	SuccessCount int32    `protobuf:"varint,2,opt,name=success_count,json=successCount" json:"success_count,omitempty"`
	FailedCount  int32    `protobuf:"varint,3,opt,name=failed_count,json=failedCount" json:"failed_count,omitempty"`
	Errors       []string `protobuf:"bytes,4,rep,name=errors" json:"errors,omitempty"`
}

func (x *BatchCreateTasksResponse) Reset()         { *x = BatchCreateTasksResponse{} }
func (x *BatchCreateTasksResponse) String() string { return proto.CompactTextString(x) }
func (*BatchCreateTasksResponse) ProtoMessage()  {}

func (x *BatchCreateTasksResponse) GetTasks() []*Task {
	if x != nil {
		return x.Tasks
	}
	return nil
}

func (x *BatchCreateTasksResponse) GetSuccessCount() int32 {
	if x != nil {
		return x.SuccessCount
	}
	return 0
}

func (x *BatchCreateTasksResponse) GetFailedCount() int32 {
	if x != nil {
		return x.FailedCount
	}
	return 0
}

func (x *BatchCreateTasksResponse) GetErrors() []string {
	if x != nil {
		return x.Errors
	}
	return nil
}

// TaskUpdateRequest 任务更新请求（双向流）
type TaskUpdateRequest struct {
	RequestId  string             `protobuf:"bytes,1,opt,name=request_id,json=requestId" json:"request_id,omitempty"`
	UpdateType string             `protobuf:"bytes,2,opt,name=update_type,json=updateType" json:"update_type,omitempty"`
	Update     *UpdateTaskRequest `protobuf:"bytes,3,opt,name=update" json:"update,omitempty"`
	Create     *CreateTaskRequest `protobuf:"bytes,4,opt,name=create" json:"create,omitempty"`
	Watch      *WatchTaskRequest  `protobuf:"bytes,5,opt,name=watch" json:"watch,omitempty"`
}

func (x *TaskUpdateRequest) Reset()         { *x = TaskUpdateRequest{} }
func (x *TaskUpdateRequest) String() string { return proto.CompactTextString(x) }
func (*TaskUpdateRequest) ProtoMessage()    {}

func (x *TaskUpdateRequest) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *TaskUpdateRequest) GetUpdateType() string {
	if x != nil {
		return x.UpdateType
	}
	return ""
}

func (x *TaskUpdateRequest) GetUpdate() *UpdateTaskRequest {
	if x != nil {
		return x.Update
	}
	return nil
}

func (x *TaskUpdateRequest) GetCreate() *CreateTaskRequest {
	if x != nil {
		return x.Create
	}
	return nil
}

func (x *TaskUpdateRequest) GetWatch() *WatchTaskRequest {
	if x != nil {
		return x.Watch
	}
	return nil
}

// TaskUpdateResponse 任务更新响应（双向流）
type TaskUpdateResponse struct {
	RequestId   string            `protobuf:"bytes,1,opt,name=request_id,json=requestId" json:"request_id,omitempty"`
	Success     bool              `protobuf:"varint,2,opt,name=success" json:"success,omitempty"`
	Error       string            `protobuf:"bytes,3,opt,name=error" json:"error,omitempty"`
	Task        *Task             `protobuf:"bytes,4,opt,name=task" json:"task,omitempty"`
	ChangeEvent *TaskChangeEvent  `protobuf:"bytes,5,opt,name=change_event,json=changeEvent" json:"change_event,omitempty"`
}

func (x *TaskUpdateResponse) Reset()         { *x = TaskUpdateResponse{} }
func (x *TaskUpdateResponse) String() string { return proto.CompactTextString(x) }
func (*TaskUpdateResponse) ProtoMessage()   {}

func (x *TaskUpdateResponse) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *TaskUpdateResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *TaskUpdateResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

func (x *TaskUpdateResponse) GetTask() *Task {
	if x != nil {
		return x.Task
	}
	return nil
}

func (x *TaskUpdateResponse) GetChangeEvent() *TaskChangeEvent {
	if x != nil {
		return x.ChangeEvent
	}
	return nil
}

func init() {
	proto.RegisterType((*WatchTaskRequest)(nil), "taskflow.WatchTaskRequest")
	proto.RegisterType((*TaskChangeEvent)(nil), "taskflow.TaskChangeEvent")
	proto.RegisterType((*BatchCreateTasksResponse)(nil), "taskflow.BatchCreateTasksResponse")
	proto.RegisterType((*TaskUpdateRequest)(nil), "taskflow.TaskUpdateRequest")
	proto.RegisterType((*TaskUpdateResponse)(nil), "taskflow.TaskUpdateResponse")
}
