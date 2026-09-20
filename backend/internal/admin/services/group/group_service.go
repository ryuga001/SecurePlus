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
	Members     []MemberSummary
}

type MemberSummary struct {
	ID    int
	Name  string
	Email string
}

type GroupListing struct {
	Items    []GroupSummary
	Page     int
	PageSize int
	Total    int64
}

const memberPreviewLimit = 3

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

	previews, err := s.repo.MemberPreviews(ctx, customerID, ids, memberPreviewLimit)
	if err != nil {
		return GroupListing{}, err
	}

	byGroupPreview := make(map[int][]MemberSummary, len(ids))
	for _, preview := range previews {
		byGroupPreview[preview.GroupID] = append(byGroupPreview[preview.GroupID], fromMemberPreview(preview))
	}

	items := make([]GroupSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, GroupSummary{
			Group:       row,
			MemberCount: byGroup[row.ID],
			Members:     byGroupPreview[row.ID],
		})
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

	previews, err := s.repo.MemberPreviews(ctx, customerID, []int{row.ID}, memberPreviewLimit)
	if err != nil {
		return GroupSummary{}, err
	}

	return GroupSummary{
		Group:       row,
		MemberCount: members,
		Members:     fromMemberPreviews(previews),
	}, nil
}

func (s *GroupService) Create(ctx context.Context, customerID int, in GroupInput) (GroupSummary, error) {
	input, err := NormalizeGroupInput(in)
	if err != nil {
		return GroupSummary{}, err
	}

	var summary GroupSummary

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r := s.repo.WithTx(tx)

		row := db.Group{CustomerID: customerID, Name: input.Name, Type: input.Type}
		if err := r.Insert(ctx, &row); err != nil {
			return groupError(err, input.Name)
		}

		if err := s.replaceMembers(ctx, r, customerID, row.ID, in.MemberIDs); err != nil {
			return err
		}

		summary = GroupSummary{Group: row}
		return nil
	})
	if err != nil {
		return GroupSummary{}, err
	}

	return s.Get(ctx, customerID, summary.Group.ID)
}

func (s *GroupService) Update(ctx context.Context, customerID, id int, in GroupInput) (GroupSummary, error) {
	input, err := NormalizeGroupInput(in)
	if err != nil {
		return GroupSummary{}, err
	}

	updates := map[string]any{"name": input.Name, "updated_at": time.Now()}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r := s.repo.WithTx(tx)

		affected, err := r.Update(ctx, customerID, id, updates)
		if err != nil {
			return groupError(err, input.Name)
		}
		if affected == 0 {
			return utils.ErrGroupNotFound
		}

		return s.replaceMembers(ctx, r, customerID, id, in.MemberIDs)
	})
	if err != nil {
		return GroupSummary{}, err
	}

	return s.Get(ctx, customerID, id)
}

func (s *GroupService) replaceMembers(ctx context.Context, r *repo.GroupRepository, customerID, groupID int, memberIDs []int) error {
	ids := uniqueIDs(memberIDs)

	var existing []int
	if len(ids) > 0 {
		var err error
		existing, err = r.FindEmailUserIDs(ctx, customerID, ids)
		if err != nil {
			return err
		}
		if len(existing) != len(ids) {
			return utils.ErrUnknownEmailUser
		}
	}

	if err := r.RemoveAllMembers(ctx, customerID, groupID); err != nil {
		return err
	}

	return r.AddMembers(ctx, customerID, groupID, existing)
}

func uniqueIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	unique := make([]int, 0, len(ids))

	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	return unique
}

func fromMemberPreviews(previews []repo.GroupMemberPreview) []MemberSummary {
	members := make([]MemberSummary, 0, len(previews))
	for _, preview := range previews {
		members = append(members, fromMemberPreview(preview))
	}
	return members
}

func fromMemberPreview(preview repo.GroupMemberPreview) MemberSummary {
	return MemberSummary{
		ID:    preview.ID,
		Name:  memberDisplayName(preview),
		Email: preview.Email,
	}
}

func memberDisplayName(preview repo.GroupMemberPreview) string {
	if name := strings.TrimSpace(preview.FirstName + " " + preview.LastName); name != "" {
		return name
	}
	return preview.Email
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
	Name      string
	Type      string
	MemberIDs []int
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
