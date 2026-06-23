package teams_entities

import "time"

type CreateTeamRequest struct {
	Name string `json:"name" binding:"required,max=255"`
}

type InviteUserRequest struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"omitempty,oneof=admin member"`
}

type TeamResponse struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy uint64    `json:"created_by"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamMemberResponse struct {
	TeamID   uint64    `json:"team_id"`
	UserID   uint64    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
