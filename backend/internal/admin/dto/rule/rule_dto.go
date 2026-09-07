package rule

import "time"

type RuleRequest struct {
	RuleName string `json:"rule_name" binding:"required,min=2,max=100"`
	Type     string `json:"type" binding:"required,oneof=REGEX KEYWORD"`
	Value    string `json:"value" binding:"required,min=1,max=1000"`
}

type RuleResponse struct {
	ID        int       `json:"id"`
	RuleName  string    `json:"rule_name"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
