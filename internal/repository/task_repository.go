package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"taskflow/internal/model"
)

// TaskRepository 任务仓储
type TaskRepository struct {
	db *SQLite
}

// NewTaskRepository 创建任务仓储
func NewTaskRepository(db *SQLite) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create 创建任务
func (r *TaskRepository) Create(task *model.Task) error {
	inputParams, _ := json.Marshal(task.InputParams)
	outputResult, _ := json.Marshal(task.OutputResult)
	dependencies, _ := json.Marshal(task.Dependencies)

	query := `INSERT INTO tasks (
		id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.DB().Exec(query,
		task.ID,
		task.Name,
		task.Description,
		task.Status,
		task.Priority,
		task.TaskType,
		string(inputParams),
		string(outputResult),
		string(dependencies),
		task.RetryCount,
		task.MaxRetries,
		task.ErrorMessage,
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
		nullableTime(task.StartedAt),
		nullableTime(task.CompletedAt),
		task.CreatedBy,
	)

	return err
}

// GetByID 根据 ID 获取任务
func (r *TaskRepository) GetByID(id string) (*model.Task, error) {
	query := `SELECT id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	FROM tasks WHERE id = ?`

	task, err := r.scanTask(r.db.DB().QueryRow(query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// 加载事件
	events, err := r.GetEventsByTaskID(id)
	if err != nil {
		return nil, err
	}
	task.Events = events

	return task, nil
}

// Update 更新任务
func (r *TaskRepository) Update(task *model.Task) error {
	inputParams, _ := json.Marshal(task.InputParams)
	outputResult, _ := json.Marshal(task.OutputResult)
	dependencies, _ := json.Marshal(task.Dependencies)

	query := `UPDATE tasks SET 
		name = ?, description = ?, status = ?, priority = ?,
		task_type = ?, input_params = ?, output_result = ?,
		dependencies = ?, retry_count = ?, max_retries = ?,
		error_message = ?, updated_at = ?, started_at = ?,
		completed_at = ?, created_by = ?
	WHERE id = ?`

	_, err := r.db.DB().Exec(query,
		task.Name,
		task.Description,
		task.Status,
		task.Priority,
		task.TaskType,
		string(inputParams),
		string(outputResult),
		string(dependencies),
		task.RetryCount,
		task.MaxRetries,
		task.ErrorMessage,
		task.UpdatedAt.Format(time.RFC3339),
		nullableTime(task.StartedAt),
		nullableTime(task.CompletedAt),
		task.CreatedBy,
		task.ID,
	)

	return err
}

// Delete 删除任务
func (r *TaskRepository) Delete(id string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := r.db.DB().Exec(query, id)
	return err
}

// List 列出任务（分页）
func (r *TaskRepository) List(limit, offset int, statusFilter *model.TaskStatus) ([]*model.Task, error) {
	query := `SELECT id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	FROM tasks`

	var args []interface{}
	if statusFilter != nil {
		query += " WHERE status = ?"
		args = append(args, *statusFilter)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.DB().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// ListByStatus 根据状态列出任务
func (r *TaskRepository) ListByStatus(status model.TaskStatus, limit int) ([]*model.Task, error) {
	return r.List(limit, 0, &status)
}

// ListByCreator 根据创建者列出任务
func (r *TaskRepository) ListByCreator(createdBy string, limit, offset int) ([]*model.Task, error) {
	query := `SELECT id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	FROM tasks WHERE created_by = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.DB().Query(query, createdBy, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// ListPending 列出待处理任务（可被调度）
func (r *TaskRepository) ListPending(limit int) ([]*model.Task, error) {
	query := `SELECT id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	FROM tasks WHERE status = ? ORDER BY priority DESC, created_at ASC LIMIT ?`

	rows, err := r.db.DB().Query(query, model.TaskStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// Count 统计任务数量
func (r *TaskRepository) Count(statusFilter *model.TaskStatus) (int, error) {
	query := "SELECT COUNT(*) FROM tasks"
	var args []interface{}
	if statusFilter != nil {
		query += " WHERE status = ?"
		args = append(args, *statusFilter)
	}

	var count int
	err := r.db.DB().QueryRow(query, args...).Scan(&count)
	return count, err
}

// AddEvent 添加任务事件
func (r *TaskRepository) AddEvent(event *model.TaskEvent) error {
	query := `INSERT INTO task_events (
		id, task_id, from_status, to_status, message, timestamp, operator
	) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.DB().Exec(query,
		event.ID,
		event.TaskID,
		event.FromStatus,
		event.ToStatus,
		event.Message,
		event.Timestamp.Format(time.RFC3339),
		event.Operator,
	)

	return err
}

// GetEventsByTaskID 获取任务的所有事件
func (r *TaskRepository) GetEventsByTaskID(taskID string) ([]model.TaskEvent, error) {
	query := `SELECT id, task_id, from_status, to_status, message, timestamp, operator
	FROM task_events WHERE task_id = ? ORDER BY timestamp ASC`

	rows, err := r.db.DB().Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.TaskEvent
	for rows.Next() {
		var event model.TaskEvent
		var timestamp string
		err := rows.Scan(
			&event.ID,
			&event.TaskID,
			&event.FromStatus,
			&event.ToStatus,
			&event.Message,
			&timestamp,
			&event.Operator,
		)
		if err != nil {
			return nil, err
		}
		event.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
		events = append(events, event)
	}

	return events, rows.Err()
}

// UpdateStatus 原子更新任务状态
func (r *TaskRepository) UpdateStatus(id string, fromStatus, toStatus model.TaskStatus) error {
	query := `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?`
	result, err := r.db.DB().Exec(query, toStatus, time.Now().Format(time.RFC3339), id, fromStatus)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("task not found or status mismatch")
	}

	return nil
}

// UpdateStatusWithEvent 原子更新任务状态并记录事件
func (r *TaskRepository) UpdateStatusWithEvent(taskID string, fromStatus, toStatus model.TaskStatus, operator, message string) error {
	return r.db.ExecTx(func(tx *sql.Tx) error {
		// 更新状态
		query := `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?`
		result, err := tx.Exec(query, toStatus, time.Now().Format(time.RFC3339), taskID, fromStatus)
		if err != nil {
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return errors.New("task not found or status mismatch")
		}

		// 添加事件
		eventID := fmt.Sprintf("%s_%d", taskID, time.Now().UnixNano())
		eventQuery := `INSERT INTO task_events (id, task_id, from_status, to_status, message, timestamp, operator)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, err = tx.Exec(eventQuery, eventID, taskID, fromStatus, toStatus, message, time.Now().Format(time.RFC3339), operator)

		return err
	})
}

// Search 搜索任务
func (r *TaskRepository) Search(keyword string, limit, offset int) ([]*model.Task, error) {
	searchPattern := "%" + keyword + "%"
	query := `SELECT id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	FROM tasks 
	WHERE name LIKE ? OR description LIKE ? OR task_type LIKE ?
	ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.DB().Query(query, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// scanTask 扫描任务行
func (r *TaskRepository) scanTask(row interface{ Scan(...interface{}) error }) (*model.Task, error) {
	var task model.Task
	var inputParams, outputResult, dependencies string
	var createdAt, updatedAt string
	var startedAt, completedAt sql.NullString

	err := row.Scan(
		&task.ID,
		&task.Name,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.TaskType,
		&inputParams,
		&outputResult,
		&dependencies,
		&task.RetryCount,
		&task.MaxRetries,
		&task.ErrorMessage,
		&createdAt,
		&updatedAt,
		&startedAt,
		&completedAt,
		&task.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if startedAt.Valid {
		task.StartedAt, _ = parseTime(startedAt.String)
	}
	if completedAt.Valid {
		task.CompletedAt, _ = parseTime(completedAt.String)
	}

	json.Unmarshal([]byte(inputParams), &task.InputParams)
	json.Unmarshal([]byte(outputResult), &task.OutputResult)
	json.Unmarshal([]byte(dependencies), &task.Dependencies)

	return &task, nil
}

// nullableTime 处理可空时间
func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// parseTime 解析时间
func parseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// BuildTaskFilter 构建任务过滤条件
type TaskFilter struct {
	Status    *model.TaskStatus
	Priority  *model.TaskPriority
	TaskType  string
	CreatedBy string
	Keyword   string
	PageSize  int
	PageIndex int
}

// ListByFilter 按条件过滤任务
func (r *TaskRepository) ListByFilter(filter TaskFilter) ([]*model.Task, int, error) {
	// 构建 WHERE 子句
	conditions := []string{}
	var args []interface{}

	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filter.Status)
	}
	if filter.Priority != nil {
		conditions = append(conditions, "priority = ?")
		args = append(args, *filter.Priority)
	}
	if filter.TaskType != "" {
		conditions = append(conditions, "task_type = ?")
		args = append(args, filter.TaskType)
	}
	if filter.CreatedBy != "" {
		conditions = append(conditions, "created_by = ?")
		args = append(args, filter.CreatedBy)
	}
	if filter.Keyword != "" {
		searchPattern := "%" + filter.Keyword + "%"
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	// 构建查询
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	var total int
	if err := r.db.DB().QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 分页参数
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageIndex < 0 {
		filter.PageIndex = 0
	}
	offset := filter.PageIndex * filter.PageSize

	// 查询列表
	listQuery := fmt.Sprintf(`SELECT id, name, description, status, priority, task_type,
		input_params, output_result, dependencies, retry_count,
		max_retries, error_message, created_at, updated_at,
		started_at, completed_at, created_by
	FROM tasks %s ORDER BY priority DESC, created_at DESC LIMIT ? OFFSET ?`, whereClause)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.DB().Query(listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}
