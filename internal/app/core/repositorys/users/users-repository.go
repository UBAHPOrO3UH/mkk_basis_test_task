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

type TeamTopTaskCreatorModel struct {
	TeamID        uint64     `gorm:"column:team_id"`
	TeamName      string     `gorm:"column:team_name"`
	TeamCreatedBy uint64     `gorm:"column:team_created_by"`
	TeamRole      string     `gorm:"column:team_role"`
	TeamJoinedAt  *time.Time `gorm:"column:team_joined_at"`
	TeamCreatedAt time.Time  `gorm:"column:team_created_at"`
	TeamUpdatedAt time.Time  `gorm:"column:team_updated_at"`
	UserID        uint64     `gorm:"column:user_id"`
	Username      string     `gorm:"column:username"`
	Name          string     `gorm:"column:name"`
	UserCreatedAt time.Time  `gorm:"column:user_created_at"`
	UserUpdatedAt time.Time  `gorm:"column:user_updated_at"`
	TaskCount     int64      `gorm:"column:task_count"`
	TeamRank      uint       `gorm:"column:team_rank"`
}

type UserRepository interface {
	Create(model *UserModel, dbConn *gorm.DB) (*UserModel, error)
	Update(id uint64, model *UserModel, dbConn *gorm.DB) (*UserModel, error)
	Delete(id uint64, dbConn *gorm.DB) (*uint64, error)

	FindByID(id uint64, dbConn *gorm.DB) (*UserModel, error)
	FindByUsername(username string, dbConn *gorm.DB) (*UserModel, error)
	FindAll(dbConn *gorm.DB) ([]*UserModel, error)
	FindAllWithFilter(params *users_entities.UserFilterRequest, dbConn *gorm.DB) (*FoundUsers, error)
	FindTopTaskCreatorsByTeamForMonth(month time.Time, dbConn *gorm.DB) ([]*TeamTopTaskCreatorModel, error)
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

func (r *UserRepositoryImpl) FindTopTaskCreatorsByTeamForMonth(
	month time.Time,
	dbConn *gorm.DB,
) ([]*TeamTopTaskCreatorModel, error) {
	models := make([]*TeamTopTaskCreatorModel, 0)
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	err := dbConn.Raw(`
		SELECT
			ranked.team_id,
			ranked.team_name,
			ranked.team_created_by,
			COALESCE(team_members.role, '') AS team_role,
			team_members.joined_at AS team_joined_at,
			ranked.team_created_at,
			ranked.team_updated_at,
			ranked.user_id,
			ranked.username,
			ranked.name,
			ranked.user_created_at,
			ranked.user_updated_at,
			ranked.task_count,
			ranked.team_rank
		FROM (
			SELECT
				tasks.team_id,
				teams.name AS team_name,
				teams.created_by AS team_created_by,
				teams.created_at AS team_created_at,
				teams.updated_at AS team_updated_at,
				users.id AS user_id,
				users.username,
				users.name,
				users.created_at AS user_created_at,
				users.updated_at AS user_updated_at,
				COUNT(*) AS task_count,
				ROW_NUMBER() OVER (
					PARTITION BY tasks.team_id
					ORDER BY COUNT(*) DESC, users.id ASC
				) AS team_rank
			FROM tasks
			JOIN teams ON teams.id = tasks.team_id
			JOIN users ON users.id = tasks.created_by
			WHERE tasks.created_at >= ? AND tasks.created_at < ?
			GROUP BY
				tasks.team_id,
				teams.name,
				teams.created_by,
				teams.created_at,
				teams.updated_at,
				users.id,
				users.username,
				users.name,
				users.created_at,
				users.updated_at
		) AS ranked
		LEFT JOIN team_members
			ON team_members.team_id = ranked.team_id
			AND team_members.user_id = ranked.user_id
		WHERE ranked.team_rank <= 3
		ORDER BY ranked.team_id ASC, ranked.team_rank ASC
	`, monthStart, monthEnd).Scan(&models).Error
	if err != nil {
		return nil, err
	}

	return models, nil
}
