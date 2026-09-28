package role

import (
	"strings"
	"time"

	repo "dpdp-backend/internal/admin/repositories/role"
	service "dpdp-backend/internal/admin/services/role"
	"dpdp-backend/internal/db"
)

type RoleListQuery struct {
	Search   string `form:"search" binding:"max=100"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type RoleRequest struct {
	Name        string   `json:"name" binding:"max=200"`
	Description string   `json:"description" binding:"max=1000"`
	Privileges  []string `json:"privileges" binding:"max=200,dive,max=100"`
}

type UserPreview struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type RoleListItem struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	System         bool          `json:"system"`
	Privileges     []string      `json:"privileges"`
	PrivilegeCount int           `json:"privilege_count"`
	Users          []UserPreview `json:"users"`
	UserCount      int           `json:"user_count"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type RoleResponse struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	System      bool          `json:"system"`
	Privileges  []string      `json:"privileges"`
	Users       []UserPreview `json:"users"`
	UserCount   int           `json:"user_count"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type RoleOption struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	System bool   `json:"system"`
}

type RoleOptionsResponse struct {
	Items []RoleOption `json:"items"`
}

type PrivilegesResponse struct {
	Privileges []string `json:"privileges"`
}

func ToRoleListItem(summary service.RoleSummary) RoleListItem {
	return RoleListItem{
		ID:             summary.Role.ID,
		Name:           summary.Role.Name,
		Description:    summary.Role.Description,
		System:         service.IsSystem(summary.Role),
		Privileges:     orEmpty(summary.Privileges),
		PrivilegeCount: summary.PrivilegeCount,
		Users:          toUserPreviews(summary.Users),
		UserCount:      summary.UserCount,
		UpdatedAt:      summary.Role.UpdatedAt,
	}
}

func ToRoleResponse(detail service.RoleDetail) RoleResponse {
	return RoleResponse{
		ID:          detail.Role.ID,
		Name:        detail.Role.Name,
		Description: detail.Role.Description,
		System:      service.IsSystem(detail.Role),
		Privileges:  orEmpty(detail.Privileges),
		Users:       toUserPreviews(detail.Users),
		UserCount:   detail.UserCount,
		CreatedAt:   detail.Role.CreatedAt,
		UpdatedAt:   detail.Role.UpdatedAt,
	}
}

func ToRoleOptions(rows []db.Role) RoleOptionsResponse {
	items := make([]RoleOption, 0, len(rows))
	for _, row := range rows {
		items = append(items, RoleOption{ID: row.ID, Name: row.Name, System: service.IsSystem(row)})
	}

	return RoleOptionsResponse{Items: items}
}

func toUserPreviews(rows []repo.UserPreview) []UserPreview {
	items := make([]UserPreview, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserPreview{
			ID:    row.ID,
			Name:  strings.TrimSpace(row.FirstName + " " + row.LastName),
			Email: row.Email,
		})
	}

	return items
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
