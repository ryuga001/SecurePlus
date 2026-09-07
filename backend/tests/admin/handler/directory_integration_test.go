//go:build integration

package handler_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	userrepo "dpdp-backend/internal/admin/repositories/emailuser"
	grouprepo "dpdp-backend/internal/admin/repositories/group"
	usersvc "dpdp-backend/internal/admin/services/emailuser"
	groupsvc "dpdp-backend/internal/admin/services/group"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

func TestIntegrationEmailUserCreateNormalises(t *testing.T) {
	h := setup(t)

	row, err := h.users.Create(context.Background(), h.tenant.ID, usersvc.EmailUserInput{
		Email:     "  Alice.Smith@Example.COM ",
		FirstName: " Alice ",
		LastName:  "  Smith ",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if row.Email != "alice.smith@example.com" || row.FirstName != "Alice" || row.LastName != "Smith" {
		t.Fatalf("row = %+v", row)
	}
}

func TestIntegrationEmailUserEmailIsGloballyUnique(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	h.user(t, h.tenant.ID, "alice@example.com")

	_, err := h.users.Create(ctx, h.tenant.ID, usersvc.EmailUserInput{
		Email:     "ALICE@example.com",
		FirstName: "Alice",
		LastName:  "Again",
	})
	if !errors.Is(err, utils.ErrEmailTaken) {
		t.Fatalf("case-insensitive duplicate = %v, want ErrEmailTaken", err)
	}

	_, err = h.users.Create(ctx, h.other.ID, usersvc.EmailUserInput{
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Elsewhere",
	})
	if !errors.Is(err, utils.ErrEmailTaken) {
		t.Fatalf("cross-tenant duplicate = %v, want ErrEmailTaken", err)
	}
}

func TestIntegrationEmailUserUpdateAndDelete(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	row := h.user(t, h.tenant.ID, "alice@example.com")

	updated, err := h.users.Update(ctx, h.tenant.ID, row.ID, usersvc.EmailUserInput{
		Email:     "alice@example.com",
		FirstName: "Alicia",
		LastName:  "Smith",
	})
	if err != nil {
		t.Fatalf("update onto its own email failed: %v", err)
	}
	if updated.FirstName != "Alicia" {
		t.Fatalf("row = %+v", updated)
	}

	if err := h.users.Delete(ctx, h.tenant.ID, row.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if err := h.users.Delete(ctx, h.tenant.ID, row.ID); !errors.Is(err, utils.ErrEmailUserNotFound) {
		t.Fatalf("second delete = %v", err)
	}
}

func TestIntegrationEmailUserTenantIsolation(t *testing.T) {
	h := setup(t)
	row := h.user(t, h.tenant.ID, "alice@example.com")
	ctx := context.Background()

	if _, err := h.users.Get(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrEmailUserNotFound) {
		t.Fatalf("get across tenants = %v", err)
	}
	if err := h.users.Delete(ctx, h.other.ID, row.ID); !errors.Is(err, utils.ErrEmailUserNotFound) {
		t.Fatalf("delete across tenants = %v", err)
	}
}

func TestIntegrationEmailUserListSearch(t *testing.T) {
	h := setup(t)

	h.user(t, h.tenant.ID, "alice@example.com")
	h.user(t, h.tenant.ID, "bob@example.com")
	h.user(t, h.other.ID, "carol@example.com")

	ctx := context.Background()

	scoped, err := h.users.List(ctx, h.tenant.ID, userrepo.ListParams{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if scoped.Total != 2 {
		t.Fatalf("total = %d, want 2", scoped.Total)
	}

	searched, err := h.users.List(ctx, h.tenant.ID, userrepo.ListParams{Search: "ALICE"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if searched.Total != 1 {
		t.Fatalf("search total = %d, want 1", searched.Total)
	}
}

func TestIntegrationGroupCreateDefaultsToUser(t *testing.T) {
	h := setup(t)

	summary, err := h.groups.Create(context.Background(), h.tenant.ID, groupsvc.GroupInput{Name: "  Finance  Team "})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if summary.Group.Type != db.GroupTypeUser || summary.Group.Name != "Finance Team" {
		t.Fatalf("group = %+v", summary.Group)
	}
}

func TestIntegrationGroupRejectsOtherTypes(t *testing.T) {
	h := setup(t)

	_, err := h.groups.Create(context.Background(), h.tenant.ID, groupsvc.GroupInput{Name: "Domains", Type: "DOMAIN"})
	if !errors.Is(err, utils.ErrInvalidGroupType) {
		t.Fatalf("error = %v, want ErrInvalidGroupType", err)
	}
}

func TestIntegrationGroupDuplicateNamePerTenant(t *testing.T) {
	h := setup(t)
	h.group(t, h.tenant.ID, "Finance")

	_, err := h.groups.Create(context.Background(), h.tenant.ID, groupsvc.GroupInput{Name: "Finance"})
	if !errors.Is(err, utils.ErrGroupNameTaken) {
		t.Fatalf("error = %v, want ErrGroupNameTaken", err)
	}

	if _, err := h.groups.Create(context.Background(), h.other.ID, groupsvc.GroupInput{Name: "Finance"}); err != nil {
		t.Fatalf("same name in another tenant must be allowed: %v", err)
	}
}

func TestIntegrationGroupMembership(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	group := h.group(t, h.tenant.ID, "Finance")
	alice := h.user(t, h.tenant.ID, "alice@example.com")
	bob := h.user(t, h.tenant.ID, "bob@example.com")

	if err := h.groups.AddMember(ctx, h.tenant.ID, group.ID, alice.ID); err != nil {
		t.Fatalf("add member failed: %v", err)
	}
	if err := h.groups.AddMember(ctx, h.tenant.ID, group.ID, bob.ID); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	if err := h.groups.AddMember(ctx, h.tenant.ID, group.ID, alice.ID); !errors.Is(err, utils.ErrMappingExists) {
		t.Fatalf("duplicate assignment = %v, want ErrMappingExists", err)
	}

	members, err := h.groups.Members(ctx, h.tenant.ID, group.ID, grouprepo.MemberListParams{})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
	if members.Total != 2 {
		t.Fatalf("members = %d, want 2", members.Total)
	}

	groups, err := h.users.Groups(ctx, h.tenant.ID, alice.ID)
	if err != nil {
		t.Fatalf("groups of user failed: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != group.ID {
		t.Fatalf("groups = %+v", groups)
	}

	summary, err := h.groups.Get(ctx, h.tenant.ID, group.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if summary.MemberCount != 2 {
		t.Fatalf("member count = %d", summary.MemberCount)
	}

	if err := h.groups.RemoveMember(ctx, h.tenant.ID, group.ID, alice.ID); err != nil {
		t.Fatalf("remove member failed: %v", err)
	}
	if err := h.groups.RemoveMember(ctx, h.tenant.ID, group.ID, alice.ID); !errors.Is(err, utils.ErrMappingNotFound) {
		t.Fatalf("second remove = %v, want ErrMappingNotFound", err)
	}
}

func TestIntegrationGroupMembershipRejectsCrossTenant(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	group := h.group(t, h.tenant.ID, "Finance")
	foreignGroup := h.group(t, h.other.ID, "Their group")
	foreignUser := h.user(t, h.other.ID, "carol@example.com")
	ownUser := h.user(t, h.tenant.ID, "alice@example.com")

	if err := h.groups.AddMember(ctx, h.tenant.ID, group.ID, foreignUser.ID); !errors.Is(err, utils.ErrUnknownEmailUser) {
		t.Fatalf("foreign user = %v, want ErrUnknownEmailUser", err)
	}

	if err := h.groups.AddMember(ctx, h.tenant.ID, foreignGroup.ID, ownUser.ID); !errors.Is(err, utils.ErrGroupNotFound) {
		t.Fatalf("foreign group = %v, want ErrGroupNotFound", err)
	}

	if total := h.count(t, &db.EmailUserGroupMapping{}, "customer_id = ?", h.tenant.ID); total != 0 {
		t.Fatalf("rejected assignments wrote %d rows", total)
	}
}

func TestIntegrationGroupDeleteRemovesMemberships(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	group := h.group(t, h.tenant.ID, "Finance")
	alice := h.user(t, h.tenant.ID, "alice@example.com")

	if err := h.groups.AddMember(ctx, h.tenant.ID, group.ID, alice.ID); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	if err := h.groups.Delete(ctx, h.tenant.ID, group.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if total := h.count(t, &db.EmailUserGroupMapping{}, "group_id = ?", group.ID); total != 0 {
		t.Fatalf("memberships = %d, want 0", total)
	}
}

func TestIntegrationEmailUserDeleteRemovesMemberships(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	group := h.group(t, h.tenant.ID, "Finance")
	alice := h.user(t, h.tenant.ID, "alice@example.com")

	if err := h.groups.AddMember(ctx, h.tenant.ID, group.ID, alice.ID); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	if err := h.users.Delete(ctx, h.tenant.ID, alice.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if total := h.count(t, &db.EmailUserGroupMapping{}, "email_user_id = ?", alice.ID); total != 0 {
		t.Fatalf("memberships = %d, want 0", total)
	}
}

func TestIntegrationGroupTenantIsolation(t *testing.T) {
	h := setup(t)
	group := h.group(t, h.tenant.ID, "Finance")
	ctx := context.Background()

	if _, err := h.groups.Get(ctx, h.other.ID, group.ID); !errors.Is(err, utils.ErrGroupNotFound) {
		t.Fatalf("get across tenants = %v", err)
	}
	if err := h.groups.Delete(ctx, h.other.ID, group.ID); !errors.Is(err, utils.ErrGroupNotFound) {
		t.Fatalf("delete across tenants = %v", err)
	}
	if _, err := h.groups.Members(ctx, h.other.ID, group.ID, grouprepo.MemberListParams{}); !errors.Is(err, utils.ErrGroupNotFound) {
		t.Fatalf("members across tenants = %v", err)
	}
}

func TestIntegrationHTTPDirectoryEndpoints(t *testing.T) {
	h := setup(t)
	router := h.router(t)

	created := do(t, router, http.MethodPost, "/api/v1/admin/email/users",
		`{"email":"Alice@Example.com","first_name":"Alice","last_name":"Smith"}`)
	expectStatus(t, created, http.StatusCreated)

	duplicate := do(t, router, http.MethodPost, "/api/v1/admin/email/users",
		`{"email":"alice@example.com","first_name":"Alice","last_name":"Smith"}`)
	expectStatus(t, duplicate, http.StatusConflict)

	if code := decodeError(t, duplicate).Error; code != utils.CodeEmailTaken {
		t.Fatalf("duplicate code = %q", code)
	}

	invalid := do(t, router, http.MethodPost, "/api/v1/admin/email/users",
		`{"email":"not-an-email","first_name":"Alice","last_name":"Smith"}`)
	expectStatus(t, invalid, http.StatusBadRequest)

	group := do(t, router, http.MethodPost, "/api/v1/admin/email/groups", `{"name":"Finance"}`)
	expectStatus(t, group, http.StatusCreated)

	badGroup := do(t, router, http.MethodPost, "/api/v1/admin/email/groups", `{"name":"Domains","type":"DOMAIN"}`)
	expectStatus(t, badGroup, http.StatusBadRequest)

	user := h.user(t, h.tenant.ID, "bob@example.com")
	groups, err := h.groups.List(context.Background(), h.tenant.ID, grouprepo.ListParams{})
	if err != nil {
		t.Fatalf("group list failed: %v", err)
	}

	groupPath := "/api/v1/admin/email/groups/" + strconv.Itoa(groups.Items[0].Group.ID)

	assigned := do(t, router, http.MethodPost, groupPath+"/users", `{"email_user_id":`+strconv.Itoa(user.ID)+`}`)
	expectStatus(t, assigned, http.StatusCreated)

	again := do(t, router, http.MethodPost, groupPath+"/users", `{"email_user_id":`+strconv.Itoa(user.ID)+`}`)
	expectStatus(t, again, http.StatusConflict)

	members := do(t, router, http.MethodGet, groupPath+"/users", "")
	expectStatus(t, members, http.StatusOK)

	removed := do(t, router, http.MethodDelete, groupPath+"/users/"+strconv.Itoa(user.ID), "")
	expectStatus(t, removed, http.StatusNoContent)
}

func TestIntegrationMigrationSeedsDirectoryPrivileges(t *testing.T) {
	h := setup(t)

	for _, prefix := range []string{"admin.email.user.%", "admin.email.group.%"} {
		var count int64
		err := h.database.Table("privileges").
			Where("name LIKE ? AND type = ?", prefix, "DASHBOARD").
			Count(&count).Error
		if err != nil {
			t.Fatalf("privilege lookup failed: %v", err)
		}

		if count != 4 {
			t.Fatalf("%s privileges = %d, want 4", prefix, count)
		}
	}
}
