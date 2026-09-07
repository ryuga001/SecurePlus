package policy

import (
	"time"

	"dpdp-backend/internal/admin/utils"
)

type PolicyRequest struct {
	PolicyName string `json:"policy_name" binding:"required,min=2,max=100"`
	Active     *bool  `json:"active"`
	GroupIDs   []int  `json:"group_ids" binding:"required,min=1,max=200,dive,min=1"`
	RuleIDs    []int  `json:"rule_ids" binding:"required,min=1,max=200,dive,min=1"`
}

type PolicyStatusRequest struct {
	Active *bool `json:"active" binding:"required"`
}

type PolicyListQuery struct {
	Search   string `form:"search" binding:"max=100"`
	Active   *bool  `form:"active"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type PolicyResponse struct {
	ID         int                   `json:"id"`
	PolicyName string                `json:"policy_name"`
	Type       string                `json:"type"`
	Active     bool                  `json:"active"`
	Groups     []utils.ReferenceItem `json:"groups"`
	Rules      []utils.ReferenceItem `json:"rules"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

type PolicyListItem struct {
	ID         int       `json:"id"`
	PolicyName string    `json:"policy_name"`
	Type       string    `json:"type"`
	Active     bool      `json:"active"`
	GroupCount int       `json:"group_count"`
	RuleCount  int       `json:"rule_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
