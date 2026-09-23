package datadiscovery

import "time"

type ConfigurationRequest struct {
	Name              string            `json:"name" binding:"required,min=2,max=100"`
	Description       string            `json:"description" binding:"max=255"`
	ConfigurationType string            `json:"configuration_type" binding:"required,oneof=MICROSOFT_ENTRA_ACCOUNT AZURE_STORAGE_ACCOUNT GOOGLE_SERVICE_ACCOUNT AWS_IAM"`
	Status            string            `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	Config            map[string]string `json:"config" binding:"omitempty,max=12"`
	Secret            map[string]string `json:"secret" binding:"omitempty,max=4"`
}

type ConfigurationUpdateRequest struct {
	Name        string            `json:"name" binding:"required,min=2,max=100"`
	Description string            `json:"description" binding:"max=255"`
	Status      string            `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	Config      map[string]string `json:"config" binding:"omitempty,max=12"`
	Secret      map[string]string `json:"secret" binding:"omitempty,max=4"`
}

type ConfigurationTestRequest struct {
	ConfigurationID   int               `json:"configuration_id" binding:"omitempty,min=1"`
	ConfigurationType string            `json:"configuration_type" binding:"omitempty,oneof=MICROSOFT_ENTRA_ACCOUNT AZURE_STORAGE_ACCOUNT GOOGLE_SERVICE_ACCOUNT AWS_IAM"`
	Config            map[string]string `json:"config" binding:"omitempty,max=12"`
	Secret            map[string]string `json:"secret" binding:"omitempty,max=4"`
}

type ConfigurationListQuery struct {
	Search             string `form:"search" binding:"max=100"`
	ConfigurationTypes string `form:"configuration_type" binding:"max=200"`
	Statuses           string `form:"status" binding:"max=100"`
	Page               int    `form:"page" binding:"omitempty,min=1"`
	PageSize           int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type ConfigurationResponse struct {
	ID                int               `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	ConfigurationType string            `json:"configuration_type"`
	Config            map[string]string `json:"config"`
	Status            string            `json:"status"`
	HasCredential     bool              `json:"has_credential"`
	PolicyCount       int64             `json:"policy_count"`
	LastTestedAt      time.Time         `json:"last_tested_at"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type ConfigurationListItem struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	ConfigurationType string    `json:"configuration_type"`
	Status            string    `json:"status"`
	PolicyCount       int64     `json:"policy_count"`
	LastTestedAt      time.Time `json:"last_tested_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type TestResponse struct {
	Status string `json:"status"`
}

type SourceCapabilityResponse struct {
	ConfigurationType string `json:"configuration_type"`
	SourceType        string `json:"source_type"`
	Label             string `json:"label"`
}
