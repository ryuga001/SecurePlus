package dashboarduser

import (
	"strings"
	"time"

	service "dpdp-backend/internal/admin/services/dashboarduser"
	"dpdp-backend/internal/db"
)

type UserListQuery struct {
	Search   string `form:"search" binding:"max=100"`
	RoleID   int    `form:"role_id" binding:"omitempty,min=1"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type UserRequest struct {
	FirstName string  `json:"first_name" binding:"max=200"`
	LastName  string  `json:"last_name" binding:"max=200"`
	Email     *string `json:"email" binding:"omitempty,max=320"`
	RoleID    int     `json:"role_id"`
}

type RoleReference struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	System bool   `json:"system"`
}

type UserResponse struct {
	ID        int            `json:"id"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Email     string         `json:"email"`
	Role      *RoleReference `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}

type CreateUserResponse struct {
	User           UserResponse `json:"user"`
	InvitationSent bool         `json:"invitation_sent"`
}

func ToUserResponse(row service.User) UserResponse {
	response := UserResponse{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		CreatedAt: row.CreatedAt,
	}

	if row.RoleID != nil && row.RoleName != nil {
		response.Role = &RoleReference{
			ID:     *row.RoleID,
			Name:   *row.RoleName,
			System: row.RoleType != nil && *row.RoleType != db.RoleTypeCustom,
		}
	}

	return response
}

func ToInput(req UserRequest) service.UserInput {
	input := service.UserInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		RoleID:    req.RoleID,
	}

	if req.Email != nil {
		input.Email = strings.TrimSpace(*req.Email)
	}

	return input
}
