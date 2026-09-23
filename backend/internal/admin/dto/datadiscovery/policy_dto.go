package datadiscovery

import (
	"time"

	"dpdp-backend/internal/admin/utils"
)

type PolicyRequest struct {
	Name            string   `json:"name" binding:"required,min=2,max=100"`
	Description     string   `json:"description" binding:"max=255"`
	ConfigurationID int      `json:"configuration_id" binding:"required,min=1"`
	SourceType      string   `json:"source_type" binding:"required,oneof=SHARE_POINT ONE_DRIVE AZURE_BLOB GOOGLE_DRIVE AWS_S3"`
	Status          string   `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	TargetList      []string `json:"target_list" binding:"required,min=1,max=200,dive,min=1,max=1024"`
	FileTypes       []string `json:"file_types" binding:"omitempty,max=200,dive,min=1,max=20"`
	RuleIDs         []int    `json:"rule_ids" binding:"required,min=1,max=200,dive,min=1"`
}

type PolicyListQuery struct {
	Search           string `form:"search" binding:"max=100"`
	SourceTypes      string `form:"source_type" binding:"max=200"`
	ConfigurationIDs string `form:"configuration_id" binding:"max=1200"`
	Statuses         string `form:"status" binding:"max=100"`
	Page             int    `form:"page" binding:"omitempty,min=1"`
	PageSize         int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type PolicyResponse struct {
	ID                int                   `json:"id"`
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	ConfigurationID   int                   `json:"configuration_id"`
	ConfigurationName string                `json:"configuration_name"`
	ConfigurationType string                `json:"configuration_type"`
	SourceType        string                `json:"source_type"`
	Status            string                `json:"status"`
	TargetList        []string              `json:"target_list"`
	FileTypes         []string              `json:"file_types"`
	Rules             []utils.ReferenceItem `json:"rules"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

type PolicyListItem struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	ConfigurationID   int       `json:"configuration_id"`
	ConfigurationName string    `json:"configuration_name"`
	ConfigurationType string    `json:"configuration_type"`
	SourceType        string    `json:"source_type"`
	Status            string    `json:"status"`
	TargetCount       int       `json:"target_count"`
	FileTypeCount     int       `json:"file_type_count"`
	RuleCount         int       `json:"rule_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
