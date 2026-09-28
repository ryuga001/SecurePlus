//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	dashboarduserhandler "dpdp-backend/internal/admin/handler/dashboarduser"
	policyhandler "dpdp-backend/internal/admin/handler/policy"
	rolehandler "dpdp-backend/internal/admin/handler/role"
	dashboarduserrepo "dpdp-backend/internal/admin/repositories/dashboarduser"
	rolerepo "dpdp-backend/internal/admin/repositories/role"
	dashboardusersvc "dpdp-backend/internal/admin/services/dashboarduser"
	rolesvc "dpdp-backend/internal/admin/services/role"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/middleware"
	"dpdp-backend/internal/notification"
	"dpdp-backend/tests/testsupport"
)

type recordingSender struct {
	mu     sync.Mutex
	emails []notification.Email
	err    error
}

func (r *recordingSender) Send(_ context.Context, email notification.Email) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.emails = append(r.emails, email)

	return r.err
}

type roleBody struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	System         bool     `json:"system"`
	Privileges     []string `json:"privileges"`
	PrivilegeCount int      `json:"privilege_count"`
	Users          []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"users"`
	UserCount int `json:"user_count"`
}

type userBody struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Role      *struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		System bool   `json:"system"`
	} `json:"role"`
}

type console struct {
	h      harness
	admin  db.Role
	actor  db.DashboardUser
	sender *recordingSender
	store  *auth.Store
}

func newConsole(t *testing.T) console {
	t.Helper()

	h := setup(t)
	admin := testsupport.SystemAdminRole(t, h.database)

	return console{
		h:      h,
		admin:  admin,
		actor:  testsupport.DashboardUser(t, h.database, h.tenant.ID, &admin.ID, address(h.tenant, "owner")),
		sender: &recordingSender{},
		store:  auth.NewStore(h.rdb),
	}
}

func address(customer db.Customer, local string) string {
	return fmt.Sprintf("%s@%s.test", local, customer.OrgName)
}

func (c console) router(t *testing.T, actor db.DashboardUser) *gin.Engine {
	t.Helper()

	roles := rolerepo.NewRoleRepository(c.h.database)
	router, group := testsupport.ActorRouter(t, actor)
	requireAdmin := middleware.RequireAdminRole(c.h.database)

	rolehandler.NewRoleHandler(rolesvc.NewRoleService(c.h.database, roles, c.store)).RegisterRoutes(group, requireAdmin)

	dashboarduserhandler.NewDashboardUserHandler(dashboardusersvc.NewDashboardUserService(
		c.h.database,
		dashboarduserrepo.NewDashboardUserRepository(c.h.database),
		roles,
		c.store,
		c.sender,
		dashboardusersvc.Hashing{Pepper: "test-pepper", Cost: 4},
		"http://localhost:3000/",
	)).RegisterRoutes(group, requireAdmin)

	return router
}

func (c console) createRole(t *testing.T, router *gin.Engine, name string, privileges ...string) roleBody {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{"name": name, "privileges": privileges})
	recorder := do(t, router, "POST", "/api/v1/admin/roles", string(payload))
	expectStatus(t, recorder, http.StatusCreated)

	return decode[roleBody](t, recorder.Body.Bytes())
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("response is not json: %v (%s)", err, raw)
	}

	return value
}

func expectError(t *testing.T, router *gin.Engine, method, path, body string, status int, code string) {
	t.Helper()

	recorder := do(t, router, method, path, body)
	expectStatus(t, recorder, status)

	if got := decodeError(t, recorder).Error; got != code {
		t.Fatalf("%s %s code = %s, want %s", method, path, got, code)
	}
}

func storedRoleID(t *testing.T, h harness, userID int) int {
	t.Helper()

	var user db.DashboardUser
	if err := h.database.Where("id = ?", userID).Take(&user).Error; err != nil {
		t.Fatalf("user lookup failed: %v", err)
	}
	if user.RoleID == nil {
		t.Fatalf("user %d has no role", userID)
	}

	return *user.RoleID
}

func TestConsoleRoutesRequireSystemAdminRole(t *testing.T) {
	c := newConsole(t)

	support := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")
	member := testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &support.ID, address(c.h.tenant, "member"))

	denied := c.router(t, member)

	for _, route := range []struct{ method, path, body string }{
		{"GET", "/api/v1/admin/roles", ""},
		{"GET", "/api/v1/admin/roles/options", ""},
		{"GET", "/api/v1/admin/privileges", ""},
		{"POST", "/api/v1/admin/roles", `{"name":"Escalated","privileges":["admin.rule.view"]}`},
		{"GET", "/api/v1/admin/users", ""},
		{"POST", "/api/v1/admin/users", `{"first_name":"A","last_name":"B","email":"x@y.test","role_id":1}`},
	} {
		recorder := do(t, denied, route.method, route.path, route.body)
		expectStatus(t, recorder, http.StatusForbidden)

		if got := decodeError(t, recorder).Error; got != "forbidden" {
			t.Fatalf("%s %s code = %s", route.method, route.path, got)
		}
	}

	allowed := c.router(t, c.actor)
	for _, path := range []string{"/api/v1/admin/roles", "/api/v1/admin/users", "/api/v1/admin/privileges", "/api/v1/admin/roles/options"} {
		expectStatus(t, do(t, allowed, "GET", path, ""), http.StatusOK)
	}
}

func TestRoleCreateStoresCustomRoleWithPrivileges(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	created := c.createRole(t, router, "  Manager  ", "admin.rule.view", "admin.policy.view", "admin.rule.view")

	if created.Name != "Manager" || created.System {
		t.Fatalf("role = %+v", created)
	}
	if !slices.Equal(created.Privileges, []string{"admin.policy.view", "admin.rule.view"}) {
		t.Fatalf("privileges = %v", created.Privileges)
	}

	var row db.Role
	if err := c.h.database.Where("id = ?", created.ID).Take(&row).Error; err != nil {
		t.Fatalf("role lookup failed: %v", err)
	}
	if row.Type != db.RoleTypeCustom || row.CustomerID != c.h.tenant.ID {
		t.Fatalf("stored role = %+v", row)
	}

	recorder := do(t, router, "GET", "/api/v1/admin/roles", "")
	expectStatus(t, recorder, http.StatusOK)

	listing := decode[utils.ListResponse[roleBody]](t, recorder.Body.Bytes())
	if len(listing.Items) != 2 || !listing.Items[0].System || listing.Items[1].ID != created.ID {
		t.Fatalf("listing = %+v", listing.Items)
	}
}

func TestRoleCreateRejectsDuplicateAndReservedNames(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	c.createRole(t, router, "Manager", "admin.rule.view")

	expectError(t, router, "POST", "/api/v1/admin/roles", `{"name":"Manager","privileges":["admin.rule.view"]}`, http.StatusConflict, utils.CodeRoleNameTaken)
	expectError(t, router, "POST", "/api/v1/admin/roles", `{"name":"ADMIN","privileges":["admin.rule.view"]}`, http.StatusConflict, utils.CodeRoleNameTaken)
}

func TestRoleCreateRejectsUnknownPrivilegeAtomically(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	expectError(t, router, "POST", "/api/v1/admin/roles", `{"name":"Broken","privileges":["admin.rule.view","admin.nope.view"]}`, http.StatusBadRequest, utils.CodePrivilegeNotFound)
	expectError(t, router, "POST", "/api/v1/admin/roles", `{"name":"Empty","privileges":[]}`, http.StatusBadRequest, utils.CodeValidation)

	if got := c.h.count(t, &db.Role{}, "customer_id = ?", c.h.tenant.ID); got != 0 {
		t.Fatalf("roles written = %d", got)
	}
}

func TestRoleUpdateReplacesPrivilegesAndDropsOnlyThatCache(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)
	ctx := context.Background()

	created := c.createRole(t, router, "Manager", "admin.rule.view")

	if err := c.store.CachePrivileges(ctx, created.ID, []string{"admin.rule.view"}, time.Hour); err != nil {
		t.Fatalf("cache seed failed: %v", err)
	}
	if err := c.store.CachePrivileges(ctx, c.admin.ID, []string{"admin.rule.view"}, time.Hour); err != nil {
		t.Fatalf("cache seed failed: %v", err)
	}

	recorder := do(t, router, "PUT", fmt.Sprintf("/api/v1/admin/roles/%d", created.ID),
		`{"name":"Managers","description":"Runs policies","privileges":["admin.policy.view","admin.policy.edit"]}`)
	expectStatus(t, recorder, http.StatusOK)

	updated := decode[roleBody](t, recorder.Body.Bytes())
	if updated.Name != "Managers" || updated.Description != "Runs policies" ||
		!slices.Equal(updated.Privileges, []string{"admin.policy.edit", "admin.policy.view"}) {
		t.Fatalf("updated = %+v", updated)
	}

	if cached, _ := c.store.Privileges(ctx, created.ID); len(cached) != 0 {
		t.Fatalf("custom role cache kept %v", cached)
	}
	if cached, _ := c.store.Privileges(ctx, c.admin.ID); len(cached) == 0 {
		t.Fatal("system admin role cache was dropped")
	}
}

func TestSystemRoleCannotBeChanged(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)
	path := fmt.Sprintf("/api/v1/admin/roles/%d", c.admin.ID)

	expectError(t, router, "PUT", path, `{"name":"Owner","privileges":["admin.rule.view"]}`, http.StatusConflict, utils.CodeSystemRoleImmutable)
	expectError(t, router, "DELETE", path, "", http.StatusConflict, utils.CodeSystemRoleImmutable)

	recorder := do(t, router, "GET", path, "")
	expectStatus(t, recorder, http.StatusOK)

	if detail := decode[roleBody](t, recorder.Body.Bytes()); !detail.System || detail.Name != c.admin.Name {
		t.Fatalf("system role = %+v", detail)
	}
}

func TestRoleDeleteIsBlockedWhileAssigned(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")
	member := testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &role.ID, address(c.h.tenant, "member"))
	path := fmt.Sprintf("/api/v1/admin/roles/%d", role.ID)

	expectError(t, router, "DELETE", path, "", http.StatusConflict, utils.CodeRoleInUse)

	if got := storedRoleID(t, c.h, member.ID); got != role.ID {
		t.Fatalf("member role = %d", got)
	}

	if err := c.h.database.Delete(&db.DashboardUser{}, member.ID).Error; err != nil {
		t.Fatalf("member delete failed: %v", err)
	}

	expectStatus(t, do(t, router, "DELETE", path, ""), http.StatusNoContent)

	if got := c.h.count(t, &db.Role{}, "id = ?", role.ID); got != 0 {
		t.Fatalf("role still exists")
	}
}

func TestRoleForeignKeyRejectsOrphaningUsers(t *testing.T) {
	c := newConsole(t)

	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")
	testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &role.ID, address(c.h.tenant, "member"))

	if err := c.h.database.Delete(&db.Role{}, role.ID).Error; !db.IsMissingReference(err) {
		t.Fatalf("direct role delete error = %v, want a foreign key violation", err)
	}
}

func TestRolesAreTenantScoped(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	foreign := testsupport.CustomRole(t, c.h.database, c.h.other.ID, "Foreign", "admin.rule.view")
	testsupport.DashboardUser(t, c.h.database, c.h.other.ID, &c.admin.ID, address(c.h.other, "owner"))

	path := fmt.Sprintf("/api/v1/admin/roles/%d", foreign.ID)
	expectError(t, router, "GET", path, "", http.StatusNotFound, utils.CodeNotFound)
	expectError(t, router, "PUT", path, `{"name":"Stolen","privileges":["admin.rule.view"]}`, http.StatusNotFound, utils.CodeNotFound)
	expectError(t, router, "DELETE", path, "", http.StatusNotFound, utils.CodeNotFound)

	recorder := do(t, router, "GET", "/api/v1/admin/roles", "")
	listing := decode[utils.ListResponse[roleBody]](t, recorder.Body.Bytes())

	for _, item := range listing.Items {
		if item.ID == foreign.ID {
			t.Fatal("another tenant's role is listed")
		}
		if item.System && (item.UserCount != 1 || len(item.Users) != 1 || item.Users[0].Email != c.actor.Email) {
			t.Fatalf("system role people leak across tenants: %+v", item)
		}
	}
}

func TestRoleListPreviewsNamesWithTotals(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support",
		"admin.rule.view", "admin.rule.edit", "admin.policy.view", "admin.policy.edit", "admin.email.group.view")

	for index := range 5 {
		testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &role.ID, address(c.h.tenant, fmt.Sprintf("member%d", index)))
	}

	recorder := do(t, router, "GET", "/api/v1/admin/roles?search=supp", "")
	expectStatus(t, recorder, http.StatusOK)

	listing := decode[utils.ListResponse[roleBody]](t, recorder.Body.Bytes())
	if len(listing.Items) != 1 {
		t.Fatalf("search returned %d roles", len(listing.Items))
	}

	item := listing.Items[0]
	if len(item.Users) != 3 || item.UserCount != 5 || len(item.Privileges) != 3 || item.PrivilegeCount != 5 {
		t.Fatalf("previews = users %d/%d privileges %d/%d", len(item.Users), item.UserCount, len(item.Privileges), item.PrivilegeCount)
	}
}

func TestUserCreateStoresSelectedRoleAndSendsInvite(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")

	for _, roleID := range []int{role.ID, c.admin.ID} {
		email := address(c.h.tenant, fmt.Sprintf("new%d", roleID))
		payload := fmt.Sprintf(`{"first_name":" Priya ","last_name":"Nair","email":"%s","role_id":%d}`, email, roleID)

		recorder := do(t, router, "POST", "/api/v1/admin/users", payload)
		expectStatus(t, recorder, http.StatusCreated)

		created := decode[struct {
			User           userBody `json:"user"`
			InvitationSent bool     `json:"invitation_sent"`
		}](t, recorder.Body.Bytes())

		if !created.InvitationSent || created.User.FirstName != "Priya" || created.User.Role == nil || created.User.Role.ID != roleID {
			t.Fatalf("created = %+v", created)
		}
		if got := storedRoleID(t, c.h, created.User.ID); got != roleID {
			t.Fatalf("stored role_id = %d, want %d", got, roleID)
		}
	}

	if len(c.sender.emails) != 2 {
		t.Fatalf("invites sent = %d", len(c.sender.emails))
	}

	invite := c.sender.emails[0]
	if invite.Template != notification.TemplateUserInvited || invite.CustomerID != c.h.tenant.ID ||
		len(invite.Vars["temp_password"]) != 20 || invite.Vars["login_url"] != "http://localhost:3000/login" {
		t.Fatalf("invite = %+v", invite)
	}
}

func TestUserCreateRejectsRolesOutsideTheTenant(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	var superAdmin db.Role
	if err := c.h.database.Where("customer_id = ? AND type = ?", db.SystemCustomerID, db.RoleTypeSuperAdmin).Take(&superAdmin).Error; err != nil {
		t.Fatalf("super admin lookup failed: %v", err)
	}

	foreign := testsupport.CustomRole(t, c.h.database, c.h.other.ID, "Foreign", "admin.rule.view")

	for _, roleID := range []int{0, superAdmin.ID, foreign.ID, 99999999} {
		payload := fmt.Sprintf(`{"first_name":"A","last_name":"B","email":"%s","role_id":%d}`, address(c.h.tenant, fmt.Sprintf("r%d", roleID)), roleID)
		expectError(t, router, "POST", "/api/v1/admin/users", payload, http.StatusBadRequest, utils.CodeRoleNotFound)
	}

	if len(c.sender.emails) != 0 {
		t.Fatal("an invite was sent for a rejected user")
	}
}

func TestUserCreateKeepsUserWhenInviteFails(t *testing.T) {
	c := newConsole(t)
	c.sender.err = errors.New("smtp down")
	router := c.router(t, c.actor)

	email := address(c.h.tenant, "late")
	payload := fmt.Sprintf(`{"first_name":"A","last_name":"B","email":"%s","role_id":%d}`, email, c.admin.ID)

	recorder := do(t, router, "POST", "/api/v1/admin/users", payload)
	expectStatus(t, recorder, http.StatusCreated)

	created := decode[struct {
		InvitationSent bool `json:"invitation_sent"`
	}](t, recorder.Body.Bytes())

	if created.InvitationSent {
		t.Fatal("invitation_sent = true after a send failure")
	}
	if got := c.h.count(t, &db.DashboardUser{}, "email = ?", email); got != 1 {
		t.Fatalf("user rows = %d", got)
	}
}

func TestUserCreateRejectsEmailUsedByAnyTenant(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	taken := address(c.h.other, "shared")
	testsupport.DashboardUser(t, c.h.database, c.h.other.ID, &c.admin.ID, taken)

	payload := fmt.Sprintf(`{"first_name":"A","last_name":"B","email":"%s","role_id":%d}`, taken, c.admin.ID)
	expectError(t, router, "POST", "/api/v1/admin/users", payload, http.StatusConflict, utils.CodeEmailTaken)
}

func TestUserUpdateRules(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)
	ctx := context.Background()

	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")
	member := testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &c.admin.ID, address(c.h.tenant, "member"))

	memberPath := fmt.Sprintf("/api/v1/admin/users/%d", member.ID)
	selfPath := fmt.Sprintf("/api/v1/admin/users/%d", c.actor.ID)

	expectError(t, router, "PUT", memberPath,
		fmt.Sprintf(`{"first_name":"A","last_name":"B","email":"new@x.test","role_id":%d}`, role.ID),
		http.StatusBadRequest, utils.CodeAdminEmailImmutable)

	expectError(t, router, "PUT", selfPath,
		fmt.Sprintf(`{"first_name":"Owner","last_name":"One","role_id":%d}`, role.ID),
		http.StatusConflict, utils.CodeSelfModification)

	expectStatus(t, do(t, router, "PUT", selfPath,
		fmt.Sprintf(`{"first_name":"Owner","last_name":"One","role_id":%d}`, c.admin.ID)), http.StatusOK)

	before, _ := c.store.Version(ctx, member.ID)

	recorder := do(t, router, "PUT", memberPath, fmt.Sprintf(`{"first_name":"Meera","last_name":"Iyer","role_id":%d}`, role.ID))
	expectStatus(t, recorder, http.StatusOK)

	if got := decode[userBody](t, recorder.Body.Bytes()); got.Role == nil || got.Role.ID != role.ID || got.FirstName != "Meera" {
		t.Fatalf("updated = %+v", got)
	}
	if after, _ := c.store.Version(ctx, member.ID); after != before {
		t.Fatalf("role change bumped the token version %d -> %d", before, after)
	}
}

func TestLastAdministratorCannotBeRemoved(t *testing.T) {
	c := newConsole(t)
	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")
	second := testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &c.admin.ID, address(c.h.tenant, "second"))

	owner := c.router(t, c.actor)
	stale := c.router(t, second)

	expectStatus(t, do(t, owner, "PUT", fmt.Sprintf("/api/v1/admin/users/%d", second.ID),
		fmt.Sprintf(`{"first_name":"A","last_name":"B","role_id":%d}`, role.ID)), http.StatusOK)

	expectError(t, stale, "PUT", fmt.Sprintf("/api/v1/admin/users/%d", c.actor.ID),
		fmt.Sprintf(`{"first_name":"A","last_name":"B","role_id":%d}`, role.ID),
		http.StatusConflict, utils.CodeLastAdministrator)

	expectError(t, stale, "DELETE", fmt.Sprintf("/api/v1/admin/users/%d", c.actor.ID), "",
		http.StatusConflict, utils.CodeLastAdministrator)
}

func TestUserDeleteRevokesSessionsAndBlocksSelf(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)
	ctx := context.Background()

	member := testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &c.admin.ID, address(c.h.tenant, "member"))

	expectError(t, router, "DELETE", fmt.Sprintf("/api/v1/admin/users/%d", c.actor.ID), "",
		http.StatusConflict, utils.CodeSelfModification)

	before, _ := c.store.Version(ctx, member.ID)

	expectStatus(t, do(t, router, "DELETE", fmt.Sprintf("/api/v1/admin/users/%d", member.ID), ""), http.StatusNoContent)

	if after, _ := c.store.Version(ctx, member.ID); after <= before {
		t.Fatalf("deleted user's token version was not bumped (%d -> %d)", before, after)
	}
	if got := c.h.count(t, &db.DashboardUser{}, "id = ?", member.ID); got != 0 {
		t.Fatal("user still exists")
	}
}

func TestUsersAreTenantScoped(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	role := testsupport.CustomRole(t, c.h.database, c.h.tenant.ID, "Support", "admin.rule.view")
	member := testsupport.DashboardUser(t, c.h.database, c.h.tenant.ID, &role.ID, address(c.h.tenant, "member"))
	outsider := testsupport.DashboardUser(t, c.h.database, c.h.other.ID, &c.admin.ID, address(c.h.other, "owner"))

	recorder := do(t, router, "GET", "/api/v1/admin/users", "")
	listing := decode[utils.ListResponse[userBody]](t, recorder.Body.Bytes())

	if listing.Total != 2 {
		t.Fatalf("total = %d, want the tenant's 2 users", listing.Total)
	}

	recorder = do(t, router, "GET", fmt.Sprintf("/api/v1/admin/users?role_id=%d", role.ID), "")
	filtered := decode[utils.ListResponse[userBody]](t, recorder.Body.Bytes())

	if filtered.Total != 1 || filtered.Items[0].ID != member.ID || filtered.Items[0].Role.Name != "Support" {
		t.Fatalf("filtered = %+v", filtered.Items)
	}

	path := fmt.Sprintf("/api/v1/admin/users/%d", outsider.ID)
	expectError(t, router, "GET", path, "", http.StatusNotFound, utils.CodeNotFound)
	expectError(t, router, "PUT", path, fmt.Sprintf(`{"first_name":"A","last_name":"B","role_id":%d}`, c.admin.ID), http.StatusNotFound, utils.CodeNotFound)
	expectError(t, router, "DELETE", path, "", http.StatusNotFound, utils.CodeNotFound)
}

func TestCustomRoleDrivesRequirePrivilege(t *testing.T) {
	c := newConsole(t)
	router := c.router(t, c.actor)

	manager := c.createRole(t, router, "Manager", "admin.policy.view")

	email := address(c.h.tenant, "manager")
	recorder := do(t, router, "POST", "/api/v1/admin/users",
		fmt.Sprintf(`{"first_name":"Ravi","last_name":"K","email":"%s","role_id":%d}`, email, manager.ID))
	expectStatus(t, recorder, http.StatusCreated)

	var user db.DashboardUser
	if err := c.h.database.Where("email = ?", email).Take(&user).Error; err != nil {
		t.Fatalf("user lookup failed: %v", err)
	}
	if user.RoleID == nil || *user.RoleID != manager.ID {
		t.Fatalf("role_id = %v, want %d", user.RoleID, manager.ID)
	}

	policies, group := testsupport.ActorRouter(t, user)
	guard := func(name string) gin.HandlerFunc { return middleware.RequirePrivilege(c.h.database, c.store, name) }
	policyhandler.NewPolicyHandler(c.h.policies).RegisterRoutes(group, guard)

	expectStatus(t, do(t, policies, "GET", "/api/v1/admin/policies", ""), http.StatusOK)
	expectForbidden(t, do(t, policies, "POST", "/api/v1/admin/policies", `{}`), "admin.policy.create")

	expectStatus(t, do(t, router, "PUT", fmt.Sprintf("/api/v1/admin/roles/%d", manager.ID),
		`{"name":"Manager","privileges":["admin.policy.view","admin.policy.create"]}`), http.StatusOK)

	if recorder := do(t, policies, "POST", "/api/v1/admin/policies", `{}`); recorder.Code == http.StatusForbidden {
		t.Fatalf("new privilege not applied after role edit: %s", recorder.Body.String())
	}
}
