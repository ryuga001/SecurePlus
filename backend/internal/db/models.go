package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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
	DKIMPrivateKey       *string    `gorm:"column:dkim_private_key" json:"-"`
	AccessTokenHash      *string    `gorm:"column:access_token_hash" json:"-"`
	AccessTokenExpiresAt *time.Time `gorm:"column:access_token_expires_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
}

func (EmailProviderConfiguration) TableName() string { return "email_provider_configurations" }

type Restriction struct {
	Mode   string   `json:"mode"`
	Values []string `json:"values"`
}

func (r Restriction) Value() (driver.Value, error) {
	if r.Mode == "" {
		r.Mode = RestrictionNone
	}
	if r.Values == nil {
		r.Values = []string{}
	}

	return json.Marshal(r)
}

func (r *Restriction) Scan(value any) error {
	if value == nil {
		*r = Restriction{Mode: RestrictionNone, Values: []string{}}
		return nil
	}

	var raw []byte

	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return errors.New("restriction column must be jsonb")
	}

	if err := json.Unmarshal(raw, r); err != nil {
		return err
	}

	if r.Values == nil {
		r.Values = []string{}
	}

	return nil
}

type Policy struct {
	ID                    int         `gorm:"column:id;primaryKey"`
	CustomerID            int         `gorm:"column:customer_id"`
	PolicyName            string      `gorm:"column:policy_name"`
	Type                  string      `gorm:"column:type"`
	Action                string      `gorm:"column:action"`
	Active                bool        `gorm:"column:active"`
	DomainRestriction     Restriction `gorm:"column:domain_restriction;type:jsonb"`
	AttachmentRestriction Restriction `gorm:"column:attachment_restriction;type:jsonb"`
	CreatedAt             time.Time   `gorm:"column:created_at"`
	UpdatedAt             time.Time   `gorm:"column:updated_at"`
}

func (Policy) TableName() string { return "policies" }

type FileType struct {
	ID        int       `gorm:"column:id;primaryKey"`
	Extension string    `gorm:"column:extension"`
	Label     string    `gorm:"column:label"`
	Active    bool      `gorm:"column:active"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (FileType) TableName() string { return "file_types" }

type Rule struct {
	ID         int       `gorm:"column:id;primaryKey"`
	CustomerID int       `gorm:"column:customer_id"`
	RuleName   string    `gorm:"column:rule_name"`
	Type       string    `gorm:"column:type"`
	Value      string    `gorm:"column:value"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (Rule) TableName() string { return "rules" }

type EmailUser struct {
	ID         int       `gorm:"column:id;primaryKey"`
	CustomerID int       `gorm:"column:customer_id"`
	Email      string    `gorm:"column:email"`
	FirstName  string    `gorm:"column:first_name"`
	LastName   string    `gorm:"column:last_name"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (EmailUser) TableName() string { return "email_users" }

type Group struct {
	ID         int       `gorm:"column:id;primaryKey"`
	CustomerID int       `gorm:"column:customer_id"`
	Name       string    `gorm:"column:name"`
	Type       string    `gorm:"column:type"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (Group) TableName() string { return "groups" }

type EmailUserGroupMapping struct {
	EmailUserID int `gorm:"column:email_user_id;primaryKey"`
	GroupID     int `gorm:"column:group_id;primaryKey"`
	CustomerID  int `gorm:"column:customer_id"`
}

func (EmailUserGroupMapping) TableName() string { return "email_user_group_mapping" }

type PolicyGroupMapping struct {
	PolicyID   int `gorm:"column:policy_id;primaryKey"`
	GroupID    int `gorm:"column:group_id;primaryKey"`
	CustomerID int `gorm:"column:customer_id"`
}

func (PolicyGroupMapping) TableName() string { return "policy_group_mapping" }

type PolicyRuleMapping struct {
	PolicyID   int `gorm:"column:policy_id;primaryKey"`
	RuleID     int `gorm:"column:rule_id;primaryKey"`
	CustomerID int `gorm:"column:customer_id"`
}

func (PolicyRuleMapping) TableName() string { return "policy_rule_mapping" }
