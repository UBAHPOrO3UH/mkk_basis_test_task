package tasks

import (
	"errors"
	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	"time"

	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

type TaskModel struct {
	ID          uint64     `gorm:"column:id;primaryKey"`
	TeamID      uint64     `gorm:"column:team_id;not null"`
	Title       string     `gorm:"column:title;not null"`
	Description *string    `gorm:"column:description"`
	Status      TaskStatus `gorm:"column:status;not null"`
	AssigneeID  *uint64    `gorm:"column:assignee_id"`
	CreatedBy   uint64     `gorm:"column:created_by;not null"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

func (*TaskModel) TableName() string {
	return "tasks"
}

type FoundTasks struct {
	Tasks        []*TaskModel
	ContentRange int64
}

type TaskRepository interface {
	Create(model *TaskModel, dbConn *gorm.DB) (*TaskModel, error)
	Update(id uint64, values map[string]interface{}, dbConn *gorm.DB) (*TaskModel, error)
	FindByID(id uint64, dbConn *gorm.DB) (*TaskModel, error)
	FindAllWithFilter(params *tasks_entities.TaskFilterRequest, dbConn *gorm.DB) (*FoundTasks, error)
}

type TaskRepositoryImpl struct{}

func NewTaskRepository() TaskRepository {
	return &TaskRepositoryImpl{}
}

func (r *TaskRepositoryImpl) Create(model *TaskModel, dbConn *gorm.DB) (*TaskModel, error) {
	if err := dbConn.Create(model).Error; err != nil {
		return nil, err
	}

	return model, nil
}

func (r *TaskRepositoryImpl) Update(
	id uint64,
	values map[string]interface{},
	dbConn *gorm.DB,
) (*TaskModel, error) {
	if err := dbConn.Model(&TaskModel{}).Where("id = ?", id).Updates(values).Error; err != nil {
		return nil, err
	}

	return r.FindByID(id, dbConn)
}

func (r *TaskRepositoryImpl) FindByID(id uint64, dbConn *gorm.DB) (*TaskModel, error) {
	var model TaskModel

	err := dbConn.Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &model, nil
}

func (r *TaskRepositoryImpl) FindAllWithFilter(
	params *tasks_entities.TaskFilterRequest,
	dbConn *gorm.DB,
) (*FoundTasks, error) {
	models := make([]*TaskModel, 0)
	var contentRange int64

	query := dbConn.Model(&TaskModel{}).Where("team_id = ?", params.TeamID)
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *params.AssigneeID)
	}

	if err := query.Count(&contentRange).Error; err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 100
	}
	if err := query.Order("id DESC").Limit(int(limit)).Offset(int(params.Shift)).Find(&models).Error; err != nil {
		return nil, err
	}

	return &FoundTasks{Tasks: models, ContentRange: contentRange}, nil
}
