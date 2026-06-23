package users

import (
	"errors"
	"fmt"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-filter"
	"strings"
	"time"

	"gorm.io/gorm"
)

type UserModel struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	Username     string    `gorm:"column:username;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null"`
	Name         string    `gorm:"column:name;not null"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (*UserModel) TableName() string {
	return "users"
}

type FoundUsers struct {
	Users        []*UserModel
	ContentRange int64
}

type UserRepository interface {
	Create(model *UserModel, dbConn *gorm.DB) (*UserModel, error)
	Update(id uint64, model *UserModel, dbConn *gorm.DB) (*UserModel, error)
	Delete(id uint64, dbConn *gorm.DB) (*uint64, error)

	FindByID(id uint64, dbConn *gorm.DB) (*UserModel, error)
	FindByUsername(username string, dbConn *gorm.DB) (*UserModel, error)
	FindAll(dbConn *gorm.DB) ([]*UserModel, error)
	FindAllWithFilter(params *users_entities.UserFilterRequest, dbConn *gorm.DB) (*FoundUsers, error)
}

type UserRepositoryImpl struct{}

func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{}
}

func (r *UserRepositoryImpl) Create(model *UserModel, dbConn *gorm.DB) (*UserModel, error) {
	result := dbConn.Create(model)
	if result.Error != nil {
		return nil, result.Error
	}

	return model, nil
}

func (r *UserRepositoryImpl) Update(
	id uint64,
	model *UserModel,
	dbConn *gorm.DB,
) (*UserModel, error) {

	result := dbConn.
		Model(&UserModel{}).
		Where("id = ?", id).
		Omit("id", "created_at").
		Updates(model)

	if result.Error != nil {
		return nil, result.Error
	}

	return r.FindByID(id, dbConn)
}

func (r *UserRepositoryImpl) Delete(id uint64, dbConn *gorm.DB) (*uint64, error) {

	result := dbConn.Delete(&UserModel{}, id)
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("user not found")
	}

	return &id, nil
}

func (r *UserRepositoryImpl) FindByID(id uint64, dbConn *gorm.DB) (*UserModel, error) {
	var model UserModel

	result := dbConn.
		Where("id = ?", id).
		First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &model, nil
}

func (r *UserRepositoryImpl) FindByUsername(username string, dbConn *gorm.DB) (*UserModel, error) {
	var model UserModel

	result := dbConn.
		Where("username = ?", username).
		First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &model, nil
}

func (r *UserRepositoryImpl) FindAll(dbConn *gorm.DB) ([]*UserModel, error) {
	var models []*UserModel

	result := dbConn.
		Order("id DESC").
		Find(&models)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return []*UserModel{}, nil
	}

	return models, nil
}

func (r *UserRepositoryImpl) FindAllWithFilter(
	params *users_entities.UserFilterRequest,
	dbConn *gorm.DB,
) (*FoundUsers, error) {
	var (
		models       []*UserModel
		contentRange int64
		limit        uint = 100
		shift        uint = 0
	)

	query := dbConn.Model(&UserModel{})

	if params != nil {
		if params.Username != "" {
			words := strings.Fields(params.Username)
			for _, word := range words {
				query = query.Where("username LIKE ?", fmt.Sprintf("%%%s%%", word))
			}
		}

		if params.Name != "" {
			words := strings.Fields(params.Name)
			for _, word := range words {
				query = query.Where("name LIKE ?", fmt.Sprintf("%%%s%%", word))
			}
		}

		if params.Limit != 0 {
			limit = params.Limit
		}

		if params.Shift != 0 {
			shift = params.Shift
		}
	}

	if err := query.Count(&contentRange).Error; err != nil {
		return nil, err
	}

	result := query.
		Order("id DESC").
		Limit(int(limit)).
		Offset(int(shift)).
		Find(&models)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		models = []*UserModel{}
	}

	return &FoundUsers{
		Users:        models,
		ContentRange: contentRange,
	}, nil
}
