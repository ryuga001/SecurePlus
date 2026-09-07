package rule

import (
	"context"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/rule"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type RuleListing struct {
	Items    []db.Rule
	Page     int
	PageSize int
	Total    int64
}

type RuleService struct {
	db   *gorm.DB
	repo *repo.RuleRepository
}

func NewRuleService(database *gorm.DB, repo *repo.RuleRepository) *RuleService {
	return &RuleService{db: database, repo: repo}
}

func (s *RuleService) List(ctx context.Context, customerID int, params repo.ListParams) (RuleListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return RuleListing{}, err
	}

	return RuleListing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *RuleService) Get(ctx context.Context, customerID, id int) (db.Rule, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return db.Rule{}, ruleError(err, "")
	}

	return row, nil
}

func (s *RuleService) Create(ctx context.Context, customerID int, in RuleInput) (db.Rule, error) {
	input, err := NormalizeRuleInput(in)
	if err != nil {
		return db.Rule{}, err
	}

	row := db.Rule{
		CustomerID: customerID,
		RuleName:   input.RuleName,
		Type:       input.Type,
		Value:      input.Value,
	}

	if err := s.repo.Insert(ctx, &row); err != nil {
		return db.Rule{}, ruleError(err, input.RuleName)
	}

	return row, nil
}

func (s *RuleService) Update(ctx context.Context, customerID, id int, in RuleInput) (db.Rule, error) {
	input, err := NormalizeRuleInput(in)
	if err != nil {
		return db.Rule{}, err
	}

	updates := map[string]any{
		"rule_name":  input.RuleName,
		"type":       input.Type,
		"value":      input.Value,
		"updated_at": time.Now(),
	}

	affected, err := s.repo.Update(ctx, customerID, id, updates)
	if err != nil {
		return db.Rule{}, ruleError(err, input.RuleName)
	}
	if affected == 0 {
		return db.Rule{}, utils.ErrRuleNotFound
	}

	return s.Get(ctx, customerID, id)
}

func (s *RuleService) Delete(ctx context.Context, customerID, id int) error {
	affected, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return ruleError(err, "")
	}
	if affected == 0 {
		return utils.ErrRuleNotFound
	}

	return nil
}

func ruleError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrRuleNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrRuleNameTaken, name)
	}

	return err
}

type RuleInput struct {
	RuleName string
	Type     string
	Value    string
}

func NormalizeRuleInput(in RuleInput) (RuleInput, error) {
	name := utils.NormalizeName(in.RuleName)
	if name == "" {
		return RuleInput{}, utils.ErrRuleNameNeeded
	}

	kind := strings.ToUpper(strings.TrimSpace(in.Type))
	if kind != db.RuleTypeRegex && kind != db.RuleTypeKeyword {
		return RuleInput{}, utils.ErrInvalidRuleType
	}

	value := in.Value
	if kind == db.RuleTypeKeyword {
		value = strings.TrimSpace(value)
	}

	if value == "" {
		return RuleInput{}, utils.ErrRuleValueNeeded
	}

	if kind == db.RuleTypeRegex {
		if _, err := regexp.Compile(value); err != nil {
			return RuleInput{}, utils.InvalidRegex(err)
		}
	}

	return RuleInput{RuleName: name, Type: kind, Value: value}, nil
}
