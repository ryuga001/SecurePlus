package emailuser

import (
	"time"

	"dpdp-backend/internal/db"
)

type EmailUserRequest struct {
	Email     string `json:"email" binding:"required,email,max=100"`
	FirstName string `json:"first_name" binding:"required,min=1,max=50"`
	LastName  string `json:"last_name" binding:"required,min=1,max=50"`
}

type EmailUserResponse struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func FromEmailUser(row db.EmailUser) EmailUserResponse {
	return EmailUserResponse{
		ID:        row.ID,
		Email:     row.Email,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func FromEmailUsers(rows []db.EmailUser) []EmailUserResponse {
	items := make([]EmailUserResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, FromEmailUser(row))
	}

	return items
}
