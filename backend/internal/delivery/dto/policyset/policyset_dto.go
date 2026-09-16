package policyset

type PolicyRuleRow struct {
	PolicyID                 int     `gorm:"column:id"`
	PolicyName               string  `gorm:"column:policy_name"`
	Action                   string  `gorm:"column:action"`
	DomainRestrictionRaw     []byte  `gorm:"column:domain_restriction"`
	AttachmentRestrictionRaw []byte  `gorm:"column:attachment_restriction"`
	EmailUserID              int     `gorm:"column:email_user_id"`
	RuleID                   *int    `gorm:"column:rule_id"`
	RuleName                 *string `gorm:"column:rule_name"`
	RuleType                 *string `gorm:"column:rule_type"`
	RuleValue                *string `gorm:"column:rule_value"`
}
