package group

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/group"
	emailuser "dpdp-backend/internal/admin/services/emailuser"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

type GroupSummary struct {
	Group       db.Group
	MemberCount int
}

type GroupListing struct {
	Items    []GroupSummary
	Page     int
	PageSize int
	Total    int64
}

type GroupService struct {
	db   *gorm.DB
	repo *repo.GroupRepository
}

func NewGroupService(database *gorm.DB, repo *repo.GroupRepository) *GroupService {
	return &GroupService{db: database, repo: repo}
}

func (s *GroupService) List(ctx context.Context, customerID int, params repo.ListParams) (GroupListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return GroupListing{}, err
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	counts, err := s.repo.MemberCounts(ctx, customerID, ids)
	if err != nil {
		return GroupListing{}, err
	}

	byGroup := make(map[int]int, len(counts))
	for _, count := range counts {
		byGroup[count.GroupID] = int(count.Total)
	}

	items := make([]GroupSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, GroupSummary{Group: row, MemberCount: byGroup[row.ID]})
	}

	return GroupListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *GroupService) Get(ctx context.Context, customerID, id int) (GroupSummary, error) {
	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return GroupSummary{}, groupError(err, "")
	}

	counts, err := s.repo.MemberCounts(ctx, customerID, []int{row.ID})
	if err != nil {
		return GroupSummary{}, err
	}

	members := 0
	if len(counts) > 0 {
		members = int(counts[0].Total)
	}

	return GroupSummary{Group: row, MemberCount: members}, nil
}

func (s *GroupService) Create(ctx context.Context, customerID int, in GroupInput) (GroupSummary, error) {
	input, err := NormalizeGroupInput(in)
	if err != nil {
		return GroupSummary{}, err
	}

	row := db.Group{CustomerID: customerID, Name: input.Name, Type: input.Type}

	if err := s.repo.Insert(ctx, &row); err != nil {
		return GroupSummary{}, groupError(err, input.Name)
	}

	return GroupSummary{Group: row}, nil
}

func (s *GroupService) Update(ctx context.Context, customerID, id int, in GroupInput) (GroupSummary, error) {
	input, err := NormalizeGroupInput(in)
	if err != nil {
		return GroupSummary{}, err
	}

	updates := map[string]any{"name": input.Name, "updated_at": time.Now()}

	affected, err := s.repo.Update(ctx, customerID, id, updates)
	if err != nil {
		return GroupSummary{}, groupError(err, input.Name)
	}
	if affected == 0 {
		return GroupSummary{}, utils.ErrGroupNotFound
	}

	return s.Get(ctx, customerID, id)
}

func (s *GroupService) Delete(ctx context.Context, customerID, id int) error {
	affected, err := s.repo.Delete(ctx, customerID, id)
	if err != nil {
		return groupError(err, "")
	}
	if affected == 0 {
		return utils.ErrGroupNotFound
	}

	return nil
}

func (s *GroupService) Members(ctx context.Context, customerID, id int, params repo.MemberListParams) (emailuser.Listing, error) {
	if _, err := s.repo.Find(ctx, customerID, id); err != nil {
		return emailuser.Listing{}, groupError(err, "")
	}

	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.Members(ctx, customerID, id, params)
	if err != nil {
		return emailuser.Listing{}, err
	}

	return emailuser.Listing{Items: rows, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *GroupService) AddMember(ctx context.Context, customerID, groupID, emailUserID int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		if _, err := repo.Find(ctx, customerID, groupID); err != nil {
			return groupError(err, "")
		}

		if _, err := repo.FindEmailUser(ctx, customerID, emailUserID); err != nil {
			if db.IsNotFound(err) {
				return utils.ErrUnknownEmailUser
			}

			return err
		}

		if err := repo.AddMember(ctx, customerID, groupID, emailUserID); err != nil {
			if db.IsDuplicate(err) {
				return utils.ErrMappingExists
			}

			return err
		}

		return nil
	})
}

func (s *GroupService) RemoveMember(ctx context.Context, customerID, groupID, emailUserID int) error {
	if _, err := s.repo.Find(ctx, customerID, groupID); err != nil {
		return groupError(err, "")
	}

	affected, err := s.repo.RemoveMember(ctx, customerID, groupID, emailUserID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return utils.ErrMappingNotFound
	}

	return nil
}

func groupError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrGroupNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrGroupNameTaken, name)
	}

	return err
}

type GroupInput struct {
	Name string
	Type string
}

func NormalizeGroupInput(in GroupInput) (GroupInput, error) {
	name := utils.NormalizeName(in.Name)
	if name == "" {
		return GroupInput{}, utils.ErrGroupNameNeeded
	}

	kind := strings.ToUpper(strings.TrimSpace(in.Type))
	if kind == "" {
		kind = db.GroupTypeUser
	}
	if kind != db.GroupTypeUser {
		return GroupInput{}, utils.ErrInvalidGroupType
	}

	return GroupInput{Name: name, Type: kind}, nil
}
