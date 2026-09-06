package db

import (
	"encoding/json"
	"time"
)

type Customer struct {
	ID        int       `gorm:"column:id;primaryKey"`
	OrgName   string    `gorm:"column:org_name"`
	JWTSecret string    `gorm:"column:jwt_secret"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Customer) TableName() string { return "customers" }

type Role struct {
	ID         int         `gorm:"column:id;primaryKey"`
	Name       string      `gorm:"column:name"`
	Type       string      `gorm:"column:type"`
	CustomerID int         `gorm:"column:customer_id"`
	Privileges []Privilege `gorm:"many2many:role_privileges;joinForeignKey:role_id;joinReferences:privilege_id"`
}

func (Role) TableName() string { return "roles" }

type Privilege struct {
	ID   int    `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
	Type string `gorm:"column:type"`
}

func (Privilege) TableName() string { return "privileges" }

type DashboardUser struct {
	ID           int       `gorm:"column:id;primaryKey"`
	CustomerID   int       `gorm:"column:customer_id"`
	RoleID       *int      `gorm:"column:role_id"`
	FirstName    string    `gorm:"column:first_name"`
	LastName     string    `gorm:"column:last_name"`
	Email        string    `gorm:"column:email"`
	PasswordHash string    `gorm:"column:password_hash"`
	PasswordSalt string    `gorm:"column:password_salt"`
	CreatedAt    time.Time `gorm:"column:created_at"`

	Customer *Customer `gorm:"foreignKey:CustomerID"`
	Role     *Role     `gorm:"foreignKey:RoleID"`
}

func (DashboardUser) TableName() string { return "dashboard_users" }

type EmailTemplate struct {
	ID         int             `gorm:"column:id;primaryKey"`
	Name       string          `gorm:"column:name"`
	Variables  json.RawMessage `gorm:"column:variables;type:jsonb"`
	Subject    string          `gorm:"column:subject"`
	Body       string          `gorm:"column:body"`
	CustomerID int             `gorm:"column:customer_id"`
	CreatedAt  time.Time       `gorm:"column:created_at"`
}

func (EmailTemplate) TableName() string { return "email_templates" }

type EmailProviderConfiguration struct {
	ID                   int        `gorm:"column:id;primaryKey"`
	CustomerID           int        `gorm:"column:customer_id"`
	Name                 string     `gorm:"column:name"`
	Domain               string     `gorm:"column:domain"`
	Provider             string     `gorm:"column:provider"`
	DKIMPublicKey        *string    `gorm:"column:dkim_public_key"`
	AccessTokenHash      *string    `gorm:"column:access_token_hash" json:"-"`
	AccessTokenExpiresAt *time.Time `gorm:"column:access_token_expires_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
}

func (EmailProviderConfiguration) TableName() string { return "email_provider_configurations" }
