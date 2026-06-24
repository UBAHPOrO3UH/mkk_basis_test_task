package teams

import (
	"errors"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	"time"

	"gorm.io/gorm"
)

type TeamModel struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name;not null"`
	CreatedBy uint64    `gorm:"column:created_by;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (*TeamModel) TableName() string {
	return "teams"
}

type TeamWithRoleModel struct {
	ID        uint64                `gorm:"column:id"`
	Name      string                `gorm:"column:name"`
	CreatedBy uint64                `gorm:"column:created_by"`
	Role      team_members.TeamRole `gorm:"column:role"`
	JoinedAt  time.Time             `gorm:"column:joined_at"`
	CreatedAt time.Time             `gorm:"column:created_at"`
	UpdatedAt time.Time             `gorm:"column:updated_at"`
}

type TeamStatsModel struct {
	ID                     uint64 `gorm:"column:id"`
	Name                   string `gorm:"column:name"`
	MemberCount            int64  `gorm:"column:member_count"`
	DoneTasksLastSevenDays int64  `gorm:"column:done_tasks_last_seven_days"`
}

type TeamRepository interface {
	Create(model *TeamModel, dbConn *gorm.DB) (*TeamModel, error)
	FindByID(id uint64, dbConn *gorm.DB) (*TeamModel, error)
	FindAllByUserID(userID uint64, dbConn *gorm.DB) ([]*TeamWithRoleModel, error)
	FindAllWithStats(dbConn *gorm.DB) ([]*TeamStatsModel, error)
}

type TeamRepositoryImpl struct{}

func NewTeamRepository() TeamRepository {
	return &TeamRepositoryImpl{}
}

func (r *TeamRepositoryImpl) Create(model *TeamModel, dbConn *gorm.DB) (*TeamModel, error) {
	if err := dbConn.Create(model).Error; err != nil {
		return nil, err
	}

	return model, nil
}

func (r *TeamRepositoryImpl) FindByID(id uint64, dbConn *gorm.DB) (*TeamModel, error) {
	var model TeamModel

	err := dbConn.Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &model, nil
}

func (r *TeamRepositoryImpl) FindAllByUserID(userID uint64, dbConn *gorm.DB) ([]*TeamWithRoleModel, error) {
	models := make([]*TeamWithRoleModel, 0)

	err := dbConn.
		Table("teams").
		Select("teams.id, teams.name, teams.created_by, team_members.role, team_members.joined_at, teams.created_at, teams.updated_at").
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ?", userID).
		Order("teams.id DESC").
		Scan(&models).Error
	if err != nil {
		return nil, err
	}

	return models, nil
}

func (r *TeamRepositoryImpl) FindAllWithStats(dbConn *gorm.DB) ([]*TeamStatsModel, error) {
	models := make([]*TeamStatsModel, 0)

	err := dbConn.
		Table("teams").
		Select(`
			teams.id,
			teams.name,
			COUNT(DISTINCT team_members.user_id) AS member_count,
			COUNT(DISTINCT CASE
				WHEN tasks.status = ?
					AND tasks.completed_at >= UTC_TIMESTAMP(3) - INTERVAL 7 DAY
				THEN tasks.id
			END) AS done_tasks_last_seven_days
		`, "done").
		Joins("LEFT JOIN team_members ON team_members.team_id = teams.id").
		Joins("LEFT JOIN tasks ON tasks.team_id = teams.id").
		Group("teams.id, teams.name").
		Order("teams.id DESC").
		Scan(&models).Error
	if err != nil {
		return nil, err
	}

	return models, nil
}
