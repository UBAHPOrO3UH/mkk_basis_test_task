package team_members

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

type TeamMemberModel struct {
	TeamID   uint64    `gorm:"column:team_id;primaryKey"`
	UserID   uint64    `gorm:"column:user_id;primaryKey"`
	Role     TeamRole  `gorm:"column:role;not null"`
	JoinedAt time.Time `gorm:"column:joined_at;autoCreateTime"`
}

func (*TeamMemberModel) TableName() string {
	return "team_members"
}

type TeamMemberRepository interface {
	Create(model *TeamMemberModel, dbConn *gorm.DB) (*TeamMemberModel, error)
	FindByTeamIDAndUserID(teamID, userID uint64, dbConn *gorm.DB) (*TeamMemberModel, error)
}

type TeamMemberRepositoryImpl struct{}

func NewTeamMemberRepository() TeamMemberRepository {
	return &TeamMemberRepositoryImpl{}
}

func (r *TeamMemberRepositoryImpl) Create(model *TeamMemberModel, dbConn *gorm.DB) (*TeamMemberModel, error) {
	if err := dbConn.Create(model).Error; err != nil {
		return nil, err
	}

	return model, nil
}

func (r *TeamMemberRepositoryImpl) FindByTeamIDAndUserID(
	teamID, userID uint64,
	dbConn *gorm.DB,
) (*TeamMemberModel, error) {
	var model TeamMemberModel

	err := dbConn.
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &model, nil
}
