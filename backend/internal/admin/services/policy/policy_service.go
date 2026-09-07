package policy

import (
	"context"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/policy"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type PolicyDetail struct {
	Policy db.Policy
	Groups []repo.PolicyReference
	Rules  []repo.PolicyReference
}

type PolicySummary struct {
	Policy     db.Policy
	GroupCount int
	RuleCount  int
}

type PolicyListing struct {
	Items    []PolicySummary
	Page     int
	PageSize int
	Total    int64
}

type PolicyService struct {
	db   *gorm.DB
	repo *repo.PolicyRepository
}

func NewPolicyService(database *gorm.DB, repo *repo.PolicyRepository) *PolicyService {
	return &PolicyService{db: database, repo: repo}
}

func (s *PolicyService) List(ctx context.Context, customerID int, params repo.ListParams) (PolicyListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return PolicyListing{}, err
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	groups, err := s.repo.GroupsFor(ctx, customerID, ids)
	if err != nil {
		return PolicyListing{}, err
	}

	rules, err := s.repo.RulesFor(ctx, customerID, ids)
	if err != nil {
		return PolicyListing{}, err
	}

	groupCounts := countByPolicy(groups)
	ruleCounts := countByPolicy(rules)

	items := make([]PolicySummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, PolicySummary{
			Policy:     row,
			GroupCount: groupCounts[row.ID],
			RuleCount:  ruleCounts[row.ID],
		})
	}

	return PolicyListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *PolicyService) Get(ctx context.Context, customerID, id int) (PolicyDetail, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return PolicyDetail{}, policyError(err, "")
	}

	return s.detail(ctx, s.repo, customerID, row)
}

func (s *PolicyService) Create(ctx context.Context, customerID int, in PolicyInput) (PolicyDetail, error) {
	input, err := NormalizePolicyInput(in)
	if err != nil {
		return PolicyDetail{}, err
	}

	row := db.Policy{
		CustomerID:            customerID,
		PolicyName:            input.PolicyName,
		Type:                  db.PolicyTypeEmail,
		Action:                input.Action,
		Active:                input.Active,
		DomainRestriction:     input.DomainRestriction,
		AttachmentRestriction: input.AttachmentRestriction,
	}

	var detail PolicyDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		if err := ensureGroups(ctx, repo, customerID, input.GroupIDs); err != nil {
			return err
		}
		if err := ensureRules(ctx, repo, customerID, input.RuleIDs); err != nil {
			return err
		}
		if err := ensureFileTypes(ctx, repo, input.AttachmentRestriction); err != nil {
			return err
		}
		if err := repo.Insert(ctx, &row); err != nil {
			return err
		}
		if err := repo.ReplaceGroups(ctx, customerID, row.ID, input.GroupIDs); err != nil {
			return err
		}
		if err := repo.ReplaceRules(ctx, customerID, row.ID, input.RuleIDs); err != nil {
			return err
		}

		detail, err = s.detail(ctx, repo, customerID, row)

		return err
	})

	if err != nil {
		return PolicyDetail{}, policyError(err, input.PolicyName)
	}

	return detail, nil
}

func (s *PolicyService) Update(ctx context.Context, customerID, id int, in PolicyInput) (PolicyDetail, error) {
	input, err := NormalizePolicyInput(in)
	if err != nil {
		return PolicyDetail{}, err
	}

	var detail PolicyDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		current, err := repo.Lock(ctx, customerID, id)
		if err != nil {
			return err
		}

		if err := ensureGroups(ctx, repo, customerID, input.GroupIDs); err != nil {
			return err
		}
		if err := ensureRules(ctx, repo, customerID, input.RuleIDs); err != nil {
			return err
		}
		if err := ensureFileTypes(ctx, repo, input.AttachmentRestriction); err != nil {
			return err
		}

		updates := map[string]any{
			"policy_name":            input.PolicyName,
			"action":                 input.Action,
			"active":                 input.Active,
			"domain_restriction":     input.DomainRestriction,
			"attachment_restriction": input.AttachmentRestriction,
			"updated_at":             time.Now(),
		}

		if _, err := repo.Update(ctx, customerID, id, updates); err != nil {
			return err
		}

		if err := repo.ReplaceGroups(ctx, customerID, id, input.GroupIDs); err != nil {
			return err
		}
		if err := repo.ReplaceRules(ctx, customerID, id, input.RuleIDs); err != nil {
			return err
		}

		current.PolicyName = input.PolicyName
		current.Action = input.Action
		current.Active = input.Active
		current.DomainRestriction = input.DomainRestriction
		current.AttachmentRestriction = input.AttachmentRestriction

		detail, err = s.detail(ctx, repo, customerID, current)

		return err
	})

	if err != nil {
		return PolicyDetail{}, policyError(err, input.PolicyName)
	}

	return detail, nil
}

func (s *PolicyService) SetActive(ctx context.Context, customerID, id int, active bool) (PolicyDetail, error) {
	updates := map[string]any{"active": active, "updated_at": time.Now()}

	affected, err := s.repo.Update(ctx, customerID, id, updates)
	if err != nil {
		return PolicyDetail{}, policyError(err, "")
	}
	if affected == 0 {
		return PolicyDetail{}, utils.ErrPolicyNotFound
	}

	return s.Get(ctx, customerID, id)
}

func (s *PolicyService) Delete(ctx context.Context, customerID, id int) error {
	affected, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return policyError(err, "")
	}
	if affected == 0 {
		return utils.ErrPolicyNotFound
	}

	return nil
}

func (s *PolicyService) detail(ctx context.Context, repo *repo.PolicyRepository, customerID int, row db.Policy) (PolicyDetail, error) {
	groups, err := repo.GroupsFor(ctx, customerID, []int{row.ID})
	if err != nil {
		return PolicyDetail{}, err
	}

	rules, err := repo.RulesFor(ctx, customerID, []int{row.ID})
	if err != nil {
		return PolicyDetail{}, err
	}

	return PolicyDetail{Policy: row, Groups: groups, Rules: rules}, nil
}

func ensureGroups(ctx context.Context, repo *repo.PolicyRepository, customerID int, ids []int) error {
	found, err := repo.LockOwnedGroupIDs(ctx, customerID, ids)
	if err != nil {
		return err
	}
	if len(found) != len(ids) {
		return utils.ErrUnknownGroup
	}

	return nil
}

func ensureRules(ctx context.Context, repo *repo.PolicyRepository, customerID int, ids []int) error {
	found, err := repo.LockOwnedRuleIDs(ctx, customerID, ids)
	if err != nil {
		return err
	}
	if len(found) != len(ids) {
		return utils.ErrUnknownRule
	}

	return nil
}

func ensureFileTypes(ctx context.Context, repository *repo.PolicyRepository, restriction db.Restriction) error {
	if len(restriction.Values) == 0 {
		return nil
	}

	found, err := repository.KnownExtensions(ctx, restriction.Values)
	if err != nil {
		return err
	}
	if len(found) != len(restriction.Values) {
		return utils.ErrInvalidRestrictionFileType
	}

	return nil
}

func (s *PolicyService) FileTypes(ctx context.Context) ([]db.FileType, error) {
	return s.repo.FileTypes(ctx)
}

func countByPolicy(rows []repo.PolicyReference) map[int]int {
	counts := make(map[int]int, len(rows))
	for _, row := range rows {
		counts[row.PolicyID]++
	}

	return counts
}

func policyError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrPolicyNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrPolicyNameTaken, name)
	case db.IsMissingReference(err):
		return utils.ErrUnknownGroup
	}

	return err
}

type PolicyInput struct {
	PolicyName            string
	Action                string
	Active                bool
	GroupIDs              []int
	RuleIDs               []int
	DomainRestriction     db.Restriction
	AttachmentRestriction db.Restriction
}

func NormalizePolicyInput(in PolicyInput) (PolicyInput, error) {
	name := utils.NormalizeName(in.PolicyName)
	if name == "" {
		return PolicyInput{}, utils.ErrPolicyNameNeeded
	}

	action := strings.ToUpper(strings.TrimSpace(in.Action))
	if action == "" {
		action = db.ActionAudit
	}
	if !validAction(action) {
		return PolicyInput{}, utils.ErrInvalidAction
	}

	groupIDs := utils.NormalizeIDs(in.GroupIDs)
	if len(groupIDs) == 0 {
		return PolicyInput{}, utils.ErrGroupsNeeded
	}

	ruleIDs := utils.NormalizeIDs(in.RuleIDs)
	if len(ruleIDs) == 0 {
		return PolicyInput{}, utils.ErrRulesNeeded
	}

	if len(groupIDs) > utils.MaxBatchIDs || len(ruleIDs) > utils.MaxBatchIDs {
		return PolicyInput{}, utils.ErrTooManyItems
	}

	domains, err := normalizeRestriction(in.DomainRestriction, normalizeDomainValue, utils.ErrInvalidRestrictionDomain)
	if err != nil {
		return PolicyInput{}, err
	}

	attachments, err := normalizeRestriction(in.AttachmentRestriction, normalizeExtensionValue, utils.ErrInvalidRestrictionFileType)
	if err != nil {
		return PolicyInput{}, err
	}

	return PolicyInput{
		PolicyName:            name,
		Action:                action,
		Active:                in.Active,
		GroupIDs:              groupIDs,
		RuleIDs:               ruleIDs,
		DomainRestriction:     domains,
		AttachmentRestriction: attachments,
	}, nil
}

func validAction(action string) bool {
	switch action {
	case db.ActionBlock, db.ActionAudit, db.ActionQuarantine, db.ActionRedact:
		return true
	}

	return false
}

func normalizeRestriction(
	in db.Restriction,
	normalizeValue func(string) string,
	invalid error,
) (db.Restriction, error) {
	mode := strings.ToUpper(strings.TrimSpace(in.Mode))
	if mode == "" {
		mode = db.RestrictionNone
	}

	if mode != db.RestrictionNone && mode != db.RestrictionBlock && mode != db.RestrictionAllow {
		return db.Restriction{}, utils.ErrInvalidRestrictionMode
	}

	if mode == db.RestrictionNone {
		return db.Restriction{Mode: db.RestrictionNone, Values: []string{}}, nil
	}

	seen := make(map[string]bool, len(in.Values))
	values := make([]string, 0, len(in.Values))

	for _, raw := range in.Values {
		value := normalizeValue(raw)
		if value == "" {
			return db.Restriction{}, invalid
		}
		if seen[value] {
			continue
		}

		seen[value] = true
		values = append(values, value)
	}

	if len(values) == 0 {
		return db.Restriction{}, utils.ErrRestrictionValuesNeeded
	}
	if len(values) > utils.MaxBatchIDs {
		return db.Restriction{}, utils.ErrTooManyItems
	}

	slices.Sort(values)

	return db.Restriction{Mode: mode, Values: values}, nil
}

func normalizeDomainValue(raw string) string {
	domain := utils.NormalizeDomain(raw)
	if !utils.ValidDomain(domain) {
		return ""
	}

	return domain
}

func normalizeExtensionValue(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimPrefix(value, ".")

	if value == "" || len(value) > 20 || strings.ContainsAny(value, " .\t/\\") {
		return ""
	}

	return value
}
