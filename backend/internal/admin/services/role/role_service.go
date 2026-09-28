package role

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	repo "dpdp-backend/internal/admin/repositories/role"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	minNameLength        = 2
	maxNameLength        = 50
	maxDescriptionLength = 255
	maxPrivileges        = 100
	previewLimit         = 3
)

type Cache interface {
	DropPrivileges(ctx context.Context, roleID int) error
	DropCustomerIdentities(ctx context.Context, customerID int) error
}

type RoleInput struct {
	Name        string
	Description string
	Privileges  []string
}

type RoleSummary struct {
	Role           db.Role
	Privileges     []string
	PrivilegeCount int
	Users          []repo.UserPreview
	UserCount      int
}

type RoleDetail struct {
	Role       db.Role
	Privileges []string
	Users      []repo.UserPreview
	UserCount  int
}

type RoleListing struct {
	Items    []RoleSummary
	Page     int
	PageSize int
	Total    int64
}

type RoleService struct {
	db    *gorm.DB
	repo  *repo.RoleRepository
	cache Cache
}

func NewRoleService(database *gorm.DB, repository *repo.RoleRepository, cache Cache) *RoleService {
	return &RoleService{db: database, repo: repository, cache: cache}
}

func IsSystem(role db.Role) bool {
	return role.Type != db.RoleTypeCustom
}

func (s *RoleService) List(ctx context.Context, customerID int, params repo.ListParams) (RoleListing, error) {
	page, pageSize := utils.NormalizePaging(params.Page, params.PageSize)

	rows, total, err := s.repo.List(ctx, customerID, params)
	if err != nil {
		return RoleListing{}, err
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	privileges, err := s.repo.PrivilegesFor(ctx, ids)
	if err != nil {
		return RoleListing{}, err
	}

	users, counts, err := s.people(ctx, s.repo, customerID, ids)
	if err != nil {
		return RoleListing{}, err
	}

	names := privilegesByRole(privileges)

	items := make([]RoleSummary, 0, len(rows))
	for _, row := range rows {
		all := names[row.ID]

		items = append(items, RoleSummary{
			Role:           row,
			Privileges:     all[:min(len(all), previewLimit)],
			PrivilegeCount: len(all),
			Users:          users[row.ID],
			UserCount:      counts[row.ID],
		})
	}

	return RoleListing{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *RoleService) Get(ctx context.Context, customerID, id int) (RoleDetail, error) {
	row, err := s.repo.FindVisible(ctx, customerID, id)
	if err != nil {
		return RoleDetail{}, roleError(err, "")
	}

	return s.detail(ctx, s.repo, customerID, row)
}

func (s *RoleService) Options(ctx context.Context, customerID int) ([]db.Role, error) {
	return s.repo.Options(ctx, customerID)
}

func (s *RoleService) Privileges(ctx context.Context) ([]string, error) {
	return s.repo.AssignablePrivileges(ctx)
}

func (s *RoleService) Create(ctx context.Context, customerID int, in RoleInput) (RoleDetail, error) {
	input, err := NormalizeRoleInput(in)
	if err != nil {
		return RoleDetail{}, err
	}

	row := db.Role{
		CustomerID:  customerID,
		Name:        input.Name,
		Description: input.Description,
		Type:        db.RoleTypeCustom,
	}

	var detail RoleDetail

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		if err := ensureNameAvailable(ctx, repository, input.Name); err != nil {
			return err
		}

		privilegeIDs, err := ensurePrivileges(ctx, repository, input.Privileges)
		if err != nil {
			return err
		}

		if err := repository.Insert(ctx, &row); err != nil {
			return err
		}
		if err := repository.ReplacePrivileges(ctx, row.ID, privilegeIDs); err != nil {
			return err
		}

		detail, err = s.detail(ctx, repository, customerID, row)

		return err
	})

	if err != nil {
		return RoleDetail{}, roleError(err, input.Name)
	}

	return detail, nil
}

func (s *RoleService) Update(ctx context.Context, customerID, id int, in RoleInput) (RoleDetail, error) {
	input, err := NormalizeRoleInput(in)
	if err != nil {
		return RoleDetail{}, err
	}

	var (
		detail  RoleDetail
		renamed bool
	)

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		current, err := repository.LockVisible(ctx, customerID, id, repo.LockUpdate)
		if err != nil {
			return err
		}
		if IsSystem(current) {
			return utils.ErrSystemRoleImmutable
		}

		if err := ensureNameAvailable(ctx, repository, input.Name); err != nil {
			return err
		}

		privilegeIDs, err := ensurePrivileges(ctx, repository, input.Privileges)
		if err != nil {
			return err
		}

		now := time.Now()

		updates := map[string]any{
			"name":        input.Name,
			"description": input.Description,
			"updated_at":  now,
		}

		if err := repository.Update(ctx, id, updates); err != nil {
			return err
		}
		if err := repository.ReplacePrivileges(ctx, id, privilegeIDs); err != nil {
			return err
		}

		renamed = current.Name != input.Name

		current.Name = input.Name
		current.Description = input.Description
		current.UpdatedAt = now

		detail, err = s.detail(ctx, repository, customerID, current)

		return err
	})

	if err != nil {
		return RoleDetail{}, roleError(err, input.Name)
	}

	s.dropPrivileges(ctx, id)

	if renamed {
		s.dropIdentities(ctx, customerID)
	}

	return detail, nil
}

func (s *RoleService) Delete(ctx context.Context, customerID, id int) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repository := s.repo.WithTx(tx)

		current, err := repository.LockVisible(ctx, customerID, id, repo.LockUpdate)
		if err != nil {
			return err
		}
		if IsSystem(current) {
			return utils.ErrSystemRoleImmutable
		}

		assigned, err := repository.CountUsers(ctx, id)
		if err != nil {
			return err
		}
		if assigned > 0 {
			return utils.ErrRoleInUse
		}

		_, err = repository.Delete(ctx, id)

		return err
	})

	if err != nil {
		return deleteRoleError(err)
	}

	s.dropPrivileges(ctx, id)

	return nil
}

func (s *RoleService) detail(ctx context.Context, repository *repo.RoleRepository, customerID int, row db.Role) (RoleDetail, error) {
	privileges, err := repository.PrivilegesFor(ctx, []int{row.ID})
	if err != nil {
		return RoleDetail{}, err
	}

	users, counts, err := s.people(ctx, repository, customerID, []int{row.ID})
	if err != nil {
		return RoleDetail{}, err
	}

	names := privilegesByRole(privileges)[row.ID]
	if names == nil {
		names = []string{}
	}

	return RoleDetail{
		Role:       row,
		Privileges: names,
		Users:      users[row.ID],
		UserCount:  counts[row.ID],
	}, nil
}

func (s *RoleService) people(ctx context.Context, repository *repo.RoleRepository, customerID int, ids []int) (map[int][]repo.UserPreview, map[int]int, error) {
	previews, err := repository.UserPreviews(ctx, customerID, ids, previewLimit)
	if err != nil {
		return nil, nil, err
	}

	totals, err := repository.UserCounts(ctx, customerID, ids)
	if err != nil {
		return nil, nil, err
	}

	users := make(map[int][]repo.UserPreview, len(ids))
	for _, preview := range previews {
		users[preview.RoleID] = append(users[preview.RoleID], preview)
	}

	counts := make(map[int]int, len(totals))
	for _, total := range totals {
		counts[total.RoleID] = total.Total
	}

	return users, counts, nil
}

func (s *RoleService) dropPrivileges(ctx context.Context, roleID int) {
	if err := s.cache.DropPrivileges(ctx, roleID); err != nil {
		slog.WarnContext(ctx, "role privilege cache invalidation failed", "role_id", roleID, "error", err)
	}
}

func (s *RoleService) dropIdentities(ctx context.Context, customerID int) {
	if err := s.cache.DropCustomerIdentities(ctx, customerID); err != nil {
		slog.WarnContext(ctx, "identity cache invalidation failed", "customer_id", customerID, "error", err)
	}
}

func ensureNameAvailable(ctx context.Context, repository *repo.RoleRepository, name string) error {
	reserved, err := repository.SystemNames(ctx)
	if err != nil {
		return err
	}

	for _, system := range reserved {
		if strings.EqualFold(system, name) {
			return utils.Taken(utils.ErrRoleNameTaken, name)
		}
	}

	return nil
}

func ensurePrivileges(ctx context.Context, repository *repo.RoleRepository, names []string) ([]int, error) {
	found, err := repository.FindPrivileges(ctx, names)
	if err != nil {
		return nil, err
	}
	if len(found) != len(names) {
		return nil, utils.ErrUnknownPrivilege
	}

	ids := make([]int, 0, len(found))
	for _, privilege := range found {
		ids = append(ids, privilege.ID)
	}

	return ids, nil
}

func privilegesByRole(rows []repo.RolePrivilege) map[int][]string {
	names := make(map[int][]string)
	for _, row := range rows {
		names[row.RoleID] = append(names[row.RoleID], row.Name)
	}

	return names
}

func roleError(err error, name string) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrRoleNotFound
	case db.IsDuplicate(err):
		return utils.Taken(utils.ErrRoleNameTaken, name)
	}

	return err
}

func deleteRoleError(err error) error {
	if db.IsMissingReference(err) {
		return utils.ErrRoleInUse
	}

	return roleError(err, "")
}

func NormalizeRoleInput(in RoleInput) (RoleInput, error) {
	name := utils.NormalizeName(in.Name)
	if length := utf8.RuneCountInString(name); length < minNameLength || length > maxNameLength {
		return RoleInput{}, utils.ErrRoleNameNeeded
	}

	description := strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return RoleInput{}, utils.ErrRoleDescriptionTooLong
	}

	privileges := make([]string, 0, len(in.Privileges))
	for _, raw := range in.Privileges {
		if value := strings.TrimSpace(raw); value != "" {
			privileges = append(privileges, value)
		}
	}

	slices.Sort(privileges)
	privileges = slices.Compact(privileges)

	if len(privileges) == 0 {
		return RoleInput{}, utils.ErrPrivilegesNeeded
	}
	if len(privileges) > maxPrivileges {
		return RoleInput{}, utils.ErrTooManyItems
	}

	return RoleInput{Name: name, Description: description, Privileges: privileges}, nil
}
