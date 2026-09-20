//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"

	grouprepo "dpdp-backend/internal/admin/repositories/group"
	groupsvc "dpdp-backend/internal/admin/services/group"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

func TestIntegrationGroupCreateSetsMembers(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	alice := h.user(t, h.tenant.ID, "alice@example.com")
	bob := h.user(t, h.tenant.ID, "bob@example.com")

	summary, err := h.groups.Create(ctx, h.tenant.ID, groupsvc.GroupInput{
		Name:      "Finance",
		MemberIDs: []int{alice.ID, bob.ID, alice.ID},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if summary.MemberCount != 2 {
		t.Fatalf("member count = %d, want 2", summary.MemberCount)
	}

	got := map[int]bool{}
	for _, member := range summary.Members {
		got[member.ID] = true
	}
	if !got[alice.ID] || !got[bob.ID] {
		t.Fatalf("preview members = %+v, want alice and bob", summary.Members)
	}

	members, err := h.groups.Members(ctx, h.tenant.ID, summary.Group.ID, grouprepo.MemberListParams{})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
	if members.Total != 2 {
		t.Fatalf("members = %d, want 2", members.Total)
	}

	memberIDs := map[int]bool{}
	for _, item := range members.Items {
		memberIDs[item.ID] = true
	}
	if !memberIDs[alice.ID] || !memberIDs[bob.ID] {
		t.Fatalf("members = %+v, want alice and bob", memberIDs)
	}
}

func TestIntegrationGroupCreateWithoutMembers(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	summary, err := h.groups.Create(ctx, h.tenant.ID, groupsvc.GroupInput{Name: "Finance"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if summary.MemberCount != 0 {
		t.Fatalf("member count = %d, want 0", summary.MemberCount)
	}

	if total := h.count(t, &db.EmailUserGroupMapping{}, "group_id = ?", summary.Group.ID); total != 0 {
		t.Fatalf("mappings = %d, want 0", total)
	}
}

func TestIntegrationGroupUpdateReplacesMembers(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	alice := h.user(t, h.tenant.ID, "alice@example.com")
	bob := h.user(t, h.tenant.ID, "bob@example.com")
	carol := h.user(t, h.tenant.ID, "carol@example.com")

	created, err := h.groups.Create(ctx, h.tenant.ID, groupsvc.GroupInput{
		Name:      "Finance",
		MemberIDs: []int{alice.ID, bob.ID},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := h.groups.Update(ctx, h.tenant.ID, created.Group.ID, groupsvc.GroupInput{
		Name:      "Finance",
		MemberIDs: []int{carol.ID},
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if updated.MemberCount != 1 {
		t.Fatalf("member count = %d, want 1", updated.MemberCount)
	}

	members, err := h.groups.Members(ctx, h.tenant.ID, created.Group.ID, grouprepo.MemberListParams{})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
	if members.Total != 1 || members.Items[0].ID != carol.ID {
		t.Fatalf("members = %+v, want only carol", members.Items)
	}
}

func TestIntegrationGroupUpdateClearsMembers(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	alice := h.user(t, h.tenant.ID, "alice@example.com")

	created, err := h.groups.Create(ctx, h.tenant.ID, groupsvc.GroupInput{
		Name:      "Finance",
		MemberIDs: []int{alice.ID},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := h.groups.Update(ctx, h.tenant.ID, created.Group.ID, groupsvc.GroupInput{Name: "Finance"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if updated.MemberCount != 0 {
		t.Fatalf("member count = %d, want 0", updated.MemberCount)
	}

	if total := h.count(t, &db.EmailUserGroupMapping{}, "group_id = ?", created.Group.ID); total != 0 {
		t.Fatalf("mappings = %d, want 0", total)
	}
}

func TestIntegrationGroupRejectsUnknownMembers(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	_, err := h.groups.Create(ctx, h.tenant.ID, groupsvc.GroupInput{
		Name:      "Finance",
		MemberIDs: []int{99999},
	})
	if !errors.Is(err, utils.ErrUnknownEmailUser) {
		t.Fatalf("error = %v, want ErrUnknownEmailUser", err)
	}

	if total := h.count(t, &db.EmailUserGroupMapping{}, "group_id IN (SELECT id FROM \"groups\" WHERE name = 'Finance')"); total != 0 {
		t.Fatalf("rejected assignment wrote %d rows", total)
	}
}

func TestIntegrationGroupRejectsForeignMembers(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	foreign := h.user(t, h.other.ID, "carol@example.com")

	_, err := h.groups.Create(ctx, h.tenant.ID, groupsvc.GroupInput{
		Name:      "Finance",
		MemberIDs: []int{foreign.ID},
	})
	if !errors.Is(err, utils.ErrUnknownEmailUser) {
		t.Fatalf("error = %v, want ErrUnknownEmailUser", err)
	}
}

func TestIntegrationHTTPGroupMemberIdsContract(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	alice := h.user(t, h.tenant.ID, "alice@example.com")
	bob := h.user(t, h.tenant.ID, "bob@example.com")

	router := h.router(t)

	created := do(t, router, http.MethodPost, "/api/v1/admin/email/groups",
		`{"name":"Finance","member_ids":[`+strconv.Itoa(alice.ID)+`,`+strconv.Itoa(bob.ID)+`]}`)
	expectStatus(t, created, http.StatusCreated)

	var createdGroup groupResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createdGroup); err != nil {
		t.Fatalf("create response is not json: %v (%s)", err, created.Body.String())
	}
	if createdGroup.MemberCount != 2 || len(createdGroup.Members) != 2 {
		t.Fatalf("create response = %+v, want 2 members", createdGroup)
	}

	members, err := h.groups.Members(ctx, h.tenant.ID, h.findGroup(t, "Finance").ID, grouprepo.MemberListParams{})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
	if members.Total != 2 {
		t.Fatalf("members = %d, want 2", members.Total)
	}

	groups, err := h.groups.List(ctx, h.tenant.ID, grouprepo.ListParams{})
	if err != nil {
		t.Fatalf("group list failed: %v", err)
	}
	if groups.Items[0].MemberCount != 2 {
		t.Fatalf("member count = %d, want 2", groups.Items[0].MemberCount)
	}
	if len(groups.Items[0].Members) != 2 {
		t.Fatalf("preview members = %+v, want 2", groups.Items[0].Members)
	}

	groupPath := "/api/v1/admin/email/groups/" + strconv.Itoa(groups.Items[0].Group.ID)

	updated := do(t, router, http.MethodPut, groupPath,
		`{"name":"Finance","member_ids":[`+strconv.Itoa(alice.ID)+`]}`)
	expectStatus(t, updated, http.StatusOK)

	membersAfter, err := h.groups.Members(ctx, h.tenant.ID, groups.Items[0].Group.ID, grouprepo.MemberListParams{})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
	if membersAfter.Total != 1 || membersAfter.Items[0].ID != alice.ID {
		t.Fatalf("members = %+v, want only alice", membersAfter.Items)
	}

	userGroups, err := h.users.Groups(ctx, h.tenant.ID, bob.ID)
	if err != nil {
		t.Fatalf("groups of user failed: %v", err)
	}
	if len(userGroups) != 0 {
		t.Fatalf("bob still in %d groups", len(userGroups))
	}
}

func (h harness) findGroup(t *testing.T, name string) db.Group {
	t.Helper()

	var row db.Group
	if err := h.database.Where("name = ?", name).First(&row).Error; err != nil {
		t.Fatalf("group %q lookup failed: %v", name, err)
	}

	return row
}

type groupResponse struct {
	ID          int                `json:"id"`
	Name        string             `json:"name"`
	MemberCount int                `json:"member_count"`
	Members     []memberResponse   `json:"members"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

type memberResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}