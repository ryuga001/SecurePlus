package policyset

import (
	"context"

	"gorm.io/gorm"

	dto "dpdp-backend/internal/delivery/engine/policy/dto/policyset"
)

const resolveQuery = `
SELECT DISTINCT
    p.id,
    p.policy_name,
    p.action,
    p.domain_restriction,
    p.attachment_restriction,
    eu.id   AS email_user_id,
    r.id    AS rule_id,
    r.rule_name,
    r.type  AS rule_type,
    r.value AS rule_value
FROM email_users AS eu
JOIN      email_user_group_mapping AS m  ON m.email_user_id = eu.id        AND m.customer_id  = eu.customer_id
JOIN      policy_group_mapping     AS pg ON pg.group_id     = m.group_id   AND pg.customer_id = m.customer_id
JOIN      policies                 AS p  ON p.id            = pg.policy_id AND p.customer_id  = pg.customer_id
LEFT JOIN policy_rule_mapping      AS pr ON pr.policy_id    = p.id         AND pr.customer_id = p.customer_id
LEFT JOIN rules                    AS r  ON r.id            = pr.rule_id   AND r.customer_id  = pr.customer_id
WHERE eu.customer_id = ?
  AND eu.email = ?
  AND p.active = true
  AND p.type = ?
ORDER BY p.id ASC, r.id ASC NULLS FIRST
`

type PolicySetRepository struct {
	db *gorm.DB
}

func NewPolicySetRepository(database *gorm.DB) *PolicySetRepository {
	return &PolicySetRepository{db: database}
}

func (r *PolicySetRepository) Resolve(ctx context.Context, customerID int, email, policyType string) ([]dto.PolicyRuleRow, error) {
	rows := make([]dto.PolicyRuleRow, 0)

	err := r.db.WithContext(ctx).
		Raw(resolveQuery, customerID, email, policyType).
		Scan(&rows).Error

	return rows, err
}
