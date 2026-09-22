package alert

import (
	"time"

	"dpdp-backend/internal/admin/utils"
)

type AlertRequest struct {
	Name             string   `json:"name" binding:"required,min=2,max=100"`
	ScheduleType     string   `json:"schedule_type" binding:"omitempty,oneof=REAL_TIME CUSTOM"`
	NotificationType string   `json:"notification_type" binding:"omitempty,oneof=EMAIL SMS"`
	Target           []string `json:"target" binding:"required,min=1,max=200,dive,min=3,max=254"`
	PolicyIDs        []int    `json:"policy_ids" binding:"required,min=1,max=200,dive,min=1"`
}

type AlertListQuery struct {
	Search           string `form:"search" binding:"max=100"`
	NotificationType string `form:"notification_type" binding:"omitempty,oneof=EMAIL SMS"`
	ScheduleType     string `form:"schedule_type" binding:"omitempty,oneof=REAL_TIME CUSTOM"`
	PolicyIDs        string `form:"policy_ids" binding:"max=1200"`
	Page             int    `form:"page" binding:"omitempty,min=1"`
	PageSize         int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type AlertResponse struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	ScheduleType     string                `json:"schedule_type"`
	NotificationType string                `json:"notification_type"`
	Target           []string              `json:"target"`
	AlertType        string                `json:"alert_type"`
	Policies         []utils.ReferenceItem `json:"policies"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type AlertListItem struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	ScheduleType     string                `json:"schedule_type"`
	NotificationType string                `json:"notification_type"`
	AlertType        string                `json:"alert_type"`
	Policies         []utils.ReferenceItem `json:"policies"`
	PolicyCount      int                   `json:"policy_count"`
	TargetCount      int                   `json:"target_count"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}
