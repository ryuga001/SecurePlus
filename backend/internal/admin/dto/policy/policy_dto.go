package policy

import (
	"time"

	"dpdp-backend/internal/admin/utils"
)

type RestrictionPayload struct {
	Mode   string   `json:"mode" binding:"omitempty,oneof=NONE BLOCK ALLOW"`
	Values []string `json:"values" binding:"omitempty,max=200,dive,min=1,max=253"`
}

type PolicyRequest struct {
	PolicyName            string              `json:"policy_name" binding:"required,min=2,max=100"`
	Action                string              `json:"action" binding:"omitempty,oneof=BLOCK AUDIT QUARANTINE REDACT"`
	Active                *bool               `json:"active"`
	GroupIDs              []int               `json:"group_ids" binding:"required,min=1,max=200,dive,min=1"`
	RuleIDs               []int               `json:"rule_ids" binding:"required,min=1,max=200,dive,min=1"`
	DomainRestriction     *RestrictionPayload `json:"domain_restriction"`
	AttachmentRestriction *RestrictionPayload `json:"attachment_restriction"`
}

type FileTypeResponse struct {
	ID        int    `json:"id"`
	Extension string `json:"extension"`
	Label     string `json:"label"`
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
	ID                    int                   `json:"id"`
	PolicyName            string                `json:"policy_name"`
	Type                  string                `json:"type"`
	Action                string                `json:"action"`
	Active                bool                  `json:"active"`
	DomainRestriction     RestrictionPayload    `json:"domain_restriction"`
	AttachmentRestriction RestrictionPayload    `json:"attachment_restriction"`
	Groups                []utils.ReferenceItem `json:"groups"`
	Rules                 []utils.ReferenceItem `json:"rules"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
}

type PolicyListItem struct {
	ID                     int       `json:"id"`
	PolicyName             string    `json:"policy_name"`
	Type                   string    `json:"type"`
	Action                 string    `json:"action"`
	Active                 bool      `json:"active"`
	DomainRestrictionMode  string    `json:"domain_restriction_mode"`
	AttachmentRestrictMode string    `json:"attachment_restriction_mode"`
	GroupCount             int       `json:"group_count"`
	RuleCount              int       `json:"rule_count"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
