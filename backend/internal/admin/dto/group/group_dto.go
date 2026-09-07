package group

import (
	"time"

	"dpdp-backend/internal/db"
)

type GroupRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
	Type string `json:"type" binding:"omitempty,oneof=USER"`
}

type AssignUserRequest struct {
	EmailUserID int `json:"email_user_id" binding:"required,min=1"`
}

type GroupResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MembershipResponse struct {
	GroupID     int `json:"group_id"`
	EmailUserID int `json:"email_user_id"`
}

func FromGroup(row db.Group, members int) GroupResponse {
	return GroupResponse{
		ID:          row.ID,
		Name:        row.Name,
		Type:        row.Type,
		MemberCount: members,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
