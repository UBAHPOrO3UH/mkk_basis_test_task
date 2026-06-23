package task_history

import (
	"time"

	"gorm.io/gorm"
)

type TaskHistoryModel struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	TaskID    uint64    `gorm:"column:task_id;not null"`
	ChangedBy uint64    `gorm:"column:changed_by;not null"`
	FieldName string    `gorm:"column:field_name;not null"`
	OldValue  *string   `gorm:"column:old_value"`
	NewValue  *string   `gorm:"column:new_value"`
	ChangedAt time.Time `gorm:"column:changed_at;autoCreateTime"`
}

func (*TaskHistoryModel) TableName() string {
	return "task_history"
}

type TaskHistoryRepository interface {
	CreateBatch(models []*TaskHistoryModel, dbConn *gorm.DB) ([]*TaskHistoryModel, error)
	FindAllByTaskID(taskID uint64, dbConn *gorm.DB) ([]*TaskHistoryModel, error)
}

type TaskHistoryRepositoryImpl struct{}

func NewTaskHistoryRepository() TaskHistoryRepository {
	return &TaskHistoryRepositoryImpl{}
}

func (r *TaskHistoryRepositoryImpl) CreateBatch(
	models []*TaskHistoryModel,
	dbConn *gorm.DB,
) ([]*TaskHistoryModel, error) {
	if len(models) == 0 {
		return models, nil
	}
	if err := dbConn.Create(&models).Error; err != nil {
		return nil, err
	}

	return models, nil
}

func (r *TaskHistoryRepositoryImpl) FindAllByTaskID(
	taskID uint64,
	dbConn *gorm.DB,
) ([]*TaskHistoryModel, error) {
	models := make([]*TaskHistoryModel, 0)

	err := dbConn.Where("task_id = ?", taskID).Order("changed_at DESC, id DESC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return models, nil
}
