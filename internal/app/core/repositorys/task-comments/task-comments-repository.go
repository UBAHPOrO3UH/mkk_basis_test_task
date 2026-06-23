package task_comments

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TaskCommentModel struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	TaskID    uint64    `gorm:"column:task_id;not null"`
	UserID    uint64    `gorm:"column:user_id;not null"`
	Body      string    `gorm:"column:body;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (*TaskCommentModel) TableName() string {
	return "task_comments"
}

type TaskCommentRepository interface {
	Create(model *TaskCommentModel, dbConn *gorm.DB) (*TaskCommentModel, error)
	Update(id uint64, body string, dbConn *gorm.DB) (*TaskCommentModel, error)
	Delete(id uint64, dbConn *gorm.DB) error
	FindByID(id uint64, dbConn *gorm.DB) (*TaskCommentModel, error)
	FindAllByTaskID(taskID uint64, dbConn *gorm.DB) ([]*TaskCommentModel, error)
}

type TaskCommentRepositoryImpl struct{}

func NewTaskCommentRepository() TaskCommentRepository {
	return &TaskCommentRepositoryImpl{}
}

func (r *TaskCommentRepositoryImpl) Create(model *TaskCommentModel, dbConn *gorm.DB) (*TaskCommentModel, error) {
	if err := dbConn.Create(model).Error; err != nil {
		return nil, err
	}

	return model, nil
}

func (r *TaskCommentRepositoryImpl) Update(id uint64, body string, dbConn *gorm.DB) (*TaskCommentModel, error) {
	if err := dbConn.Model(&TaskCommentModel{}).Where("id = ?", id).Update("body", body).Error; err != nil {
		return nil, err
	}

	return r.FindByID(id, dbConn)
}

func (r *TaskCommentRepositoryImpl) Delete(id uint64, dbConn *gorm.DB) error {
	return dbConn.Delete(&TaskCommentModel{}, id).Error
}

func (r *TaskCommentRepositoryImpl) FindByID(id uint64, dbConn *gorm.DB) (*TaskCommentModel, error) {
	var model TaskCommentModel

	err := dbConn.Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &model, nil
}

func (r *TaskCommentRepositoryImpl) FindAllByTaskID(
	taskID uint64,
	dbConn *gorm.DB,
) ([]*TaskCommentModel, error) {
	models := make([]*TaskCommentModel, 0)

	err := dbConn.Where("task_id = ?", taskID).Order("created_at ASC, id ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return models, nil
}
