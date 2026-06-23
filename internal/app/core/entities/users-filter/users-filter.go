package users_filter

type UserFilterRequest struct {
	Username string `form:"username" binding:"omitempty,max=255"`
	Name     string `form:"name" binding:"omitempty,max=255"`

	Limit uint `form:"limit" binding:"omitempty,min=1,max=100"`
	Shift uint `form:"shift" binding:"omitempty,min=0"`
}
