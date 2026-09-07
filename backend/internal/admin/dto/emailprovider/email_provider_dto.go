package emailprovider

import "time"

type ConfigurationRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Domain   string `json:"domain" binding:"required,max=253"`
	Provider string `json:"provider" binding:"required,max=20"`
}

type ConfigurationResponse struct {
	ID                   int        `json:"id"`
	Name                 string     `json:"name"`
	Domain               string     `json:"domain"`
	Provider             string     `json:"provider"`
	DKIMPublicKey        *string    `json:"dkim_public_key"`
	HasAccessToken       bool       `json:"has_access_token"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type DKIMResponse struct {
	Selector      string `json:"selector"`
	RecordName    string `json:"record_name"`
	DKIMPublicKey string `json:"dkim_public_key"`
}

type AccessTokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}
