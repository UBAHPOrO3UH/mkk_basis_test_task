package tasks_entities

import "time"

type CreateTaskRequest struct {
	TeamID      uint64  `json:"team_id" binding:"required"`
	Title       string  `json:"title" binding:"required,max=255"`
	Description *string `json:"description"`
	AssigneeID  *uint64 `json:"assignee_id"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=255"`
	Description *string `json:"description"`
	Status      *string `json:"status" binding:"omitempty,oneof=todo in_progress done"`
	AssigneeID  *uint64 `json:"assignee_id"`
}

type TaskFilterRequest struct {
	TeamID     uint64  `form:"team_id" binding:"required"`
	Status     string  `form:"status" binding:"omitempty,oneof=todo in_progress done"`
	AssigneeID *uint64 `form:"assignee_id"`
	Limit      uint    `form:"limit" binding:"omitempty,max=100"`
	Shift      uint    `form:"shift"`
}

type TaskResponse struct {
	ID          uint64     `json:"id"`
	TeamID      uint64     `json:"team_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	AssigneeID  *uint64    `json:"assignee_id"`
	CreatedBy   uint64     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type TaskHistoryResponse struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	ChangedBy uint64    `json:"changed_by"`
	FieldName string    `json:"field_name"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}
