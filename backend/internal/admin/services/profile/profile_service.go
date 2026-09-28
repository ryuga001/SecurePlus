package profile

import (
	"context"
	"errors"
	"log/slog"
	"unicode/utf8"

	repo "dpdp-backend/internal/admin/repositories/profile"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const PrivilegeOrganizationEdit = "admin.organization.edit"

const (
	minOrgNameLength = 2
	maxOrgNameLength = 100
	maxNameLength    = 50
)

type IdentityCache interface {
	DropIdentity(ctx context.Context, userID int) error
	DropCustomerIdentities(ctx context.Context, customerID int) error
}

type PrivilegeLookup interface {
	Privileges(ctx context.Context, roleID int) ([]string, error)
}

type Actor struct {
	CustomerID int
	UserID     int
	RoleID     *int
}

type Profile struct {
	OrgName    string
	FirstName  string
	LastName   string
	AdminEmail string
}

type ProfileInput struct {
	OrgName   *string
	FirstName *string
	LastName  *string
}

type ProfileService struct {
	repo          *repo.ProfileRepository
	privileges    PrivilegeLookup
	identityCache IdentityCache
}

func NewProfileService(repository *repo.ProfileRepository, privileges PrivilegeLookup, identityCache IdentityCache) *ProfileService {
	return &ProfileService{
		repo:          repository,
		privileges:    privileges,
		identityCache: identityCache,
	}
}

func (s *ProfileService) Get(ctx context.Context, actor Actor) (Profile, error) {
	row, err := s.repo.Find(ctx, actor.CustomerID, actor.UserID)
	if err != nil {
		return Profile{}, profileError(err)
	}

	return toProfile(row), nil
}

func (s *ProfileService) Update(ctx context.Context, actor Actor, input ProfileInput) (Profile, error) {
	normalized, err := NormalizeProfileInput(input)
	if err != nil {
		return Profile{}, err
	}

	current, err := s.repo.Find(ctx, actor.CustomerID, actor.UserID)
	if err != nil {
		return Profile{}, profileError(err)
	}

	changes := pendingChanges(current, normalized)
	if changes == (repo.Changes{}) {
		return toProfile(current), nil
	}

	if changes.OrgName != nil {
		if err := s.authorizeOrganizationEdit(ctx, actor); err != nil {
			return Profile{}, err
		}
	}

	updated, err := s.repo.Update(ctx, actor.CustomerID, actor.UserID, changes)
	if err != nil {
		return Profile{}, profileError(err)
	}

	s.invalidate(ctx, actor, changes.OrgName != nil)

	return toProfile(updated), nil
}

func (s *ProfileService) authorizeOrganizationEdit(ctx context.Context, actor Actor) error {
	if actor.RoleID == nil {
		return utils.ErrOrgEditForbidden
	}

	granted, err := s.privileges.Privileges(ctx, *actor.RoleID)
	if err != nil {
		slog.ErrorContext(ctx, "privilege lookup failed", "role_id", *actor.RoleID, "error", err)
		return utils.ErrUnavailable
	}

	for _, name := range granted {
		if name == PrivilegeOrganizationEdit {
			return nil
		}
	}

	return utils.ErrOrgEditForbidden
}

func (s *ProfileService) invalidate(ctx context.Context, actor Actor, organizationChanged bool) {
	var err error

	if organizationChanged {
		err = s.identityCache.DropCustomerIdentities(ctx, actor.CustomerID)
	} else {
		err = s.identityCache.DropIdentity(ctx, actor.UserID)
	}

	if err != nil {
		slog.WarnContext(ctx, "profile cache invalidation failed", "customer_id", actor.CustomerID, "user_id", actor.UserID, "error", err)
	}
}

func NormalizeProfileInput(input ProfileInput) (ProfileInput, error) {
	org, err := normalizeOrgName(input.OrgName)
	if err != nil {
		return ProfileInput{}, err
	}

	first, err := normalizePersonName(input.FirstName)
	if err != nil {
		return ProfileInput{}, err
	}

	last, err := normalizePersonName(input.LastName)
	if err != nil {
		return ProfileInput{}, err
	}

	return ProfileInput{OrgName: org, FirstName: first, LastName: last}, nil
}

func normalizeOrgName(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}

	name := utils.NormalizeName(*raw)
	length := utf8.RuneCountInString(name)

	if length < minOrgNameLength || length > maxOrgNameLength {
		return nil, utils.ErrOrgNameNeeded
	}

	return &name, nil
}

func normalizePersonName(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}

	name := utils.NormalizeName(*raw)

	if name == "" {
		return nil, utils.ErrNameNeeded
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return nil, utils.ErrNameTooLong
	}

	return &name, nil
}

func pendingChanges(current repo.ProfileRow, input ProfileInput) repo.Changes {
	var changes repo.Changes

	if input.OrgName != nil && *input.OrgName != current.OrgName {
		changes.OrgName = input.OrgName
	}
	if input.FirstName != nil && *input.FirstName != current.FirstName {
		changes.FirstName = input.FirstName
	}
	if input.LastName != nil && *input.LastName != current.LastName {
		changes.LastName = input.LastName
	}

	return changes
}

func toProfile(row repo.ProfileRow) Profile {
	return Profile{
		OrgName:    row.OrgName,
		FirstName:  row.FirstName,
		LastName:   row.LastName,
		AdminEmail: row.Email,
	}
}

func profileError(err error) error {
	switch {
	case db.IsNotFound(err):
		return utils.ErrProfileNotFound
	case errors.Is(err, repo.ErrOrgNameConflict):
		return utils.ErrOrgNameTaken
	}

	return err
}
