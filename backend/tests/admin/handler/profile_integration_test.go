//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	profilehandler "dpdp-backend/internal/admin/handler/profile"
	profilerepo "dpdp-backend/internal/admin/repositories/profile"
	profilesvc "dpdp-backend/internal/admin/services/profile"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/db"
	"dpdp-backend/tests/testsupport"
)

type profileBody struct {
	OrgName    string `json:"org_name"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	AdminEmail string `json:"admin_email"`
}

type rolePrivileges struct {
	database *gorm.DB
	store    *auth.Store
}

func (r rolePrivileges) Privileges(ctx context.Context, roleID int) ([]string, error) {
	return auth.PrivilegeNames(ctx, r.database, r.store, roleID)
}

func dashboardUser(t *testing.T, h harness, customerID int, roleID *int, email string) db.DashboardUser {
	t.Helper()

	return testsupport.DashboardUser(t, h.database, customerID, roleID, email)
}

func profileRouter(t *testing.T, h harness, user db.DashboardUser) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	store := auth.NewStore(h.rdb)
	service := profilesvc.NewProfileService(
		profilerepo.NewProfileRepository(h.database),
		rolePrivileges{database: h.database, store: store},
		store,
	)

	router := gin.New()
	group := router.Group("/api/v1")

	group.Use(func(c *gin.Context) {
		auth.SetClaims(c, &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.Itoa(user.ID)},
			CustomerID:       user.CustomerID,
			RoleID:           user.RoleID,
		})
		c.Next()
	})

	profilehandler.NewProfileHandler(service).RegisterRoutes(group, nil)

	return router
}

func profileUser(t *testing.T, h harness, privileges ...string) db.DashboardUser {
	t.Helper()

	role := testsupport.Role(t, h.database, h.tenant.ID, privileges...)

	return dashboardUser(t, h, h.tenant.ID, &role.ID, "owner-"+strconv.Itoa(role.ID)+"@acme.test")
}

func decodeProfile(t *testing.T, raw []byte) profileBody {
	t.Helper()

	var payload profileBody
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("response is not a profile: %v (%s)", err, raw)
	}

	return payload
}

func orgName(t *testing.T, h harness, customerID int) string {
	t.Helper()

	var customer db.Customer
	if err := h.database.Where("id = ?", customerID).Take(&customer).Error; err != nil {
		t.Fatalf("customer lookup failed: %v", err)
	}

	return customer.OrgName
}

func storedUser(t *testing.T, h harness, id int) db.DashboardUser {
	t.Helper()

	var user db.DashboardUser
	if err := h.database.Where("id = ?", id).Take(&user).Error; err != nil {
		t.Fatalf("dashboard user lookup failed: %v", err)
	}

	return user
}

func TestProfileReadReturnsCallerAndOrganization(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h)

	recorder := do(t, profileRouter(t, h, user), "GET", "/api/v1/admin/profile", "")
	expectStatus(t, recorder, 200)

	got := decodeProfile(t, recorder.Body.Bytes())
	want := profileBody{OrgName: h.tenant.OrgName, FirstName: "Asha", LastName: "Rao", AdminEmail: user.Email}

	if got != want {
		t.Fatalf("profile = %+v, want %+v", got, want)
	}
}

func TestProfileUpdatesNamesWithoutOrganizationPrivilege(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h)

	recorder := do(t, profileRouter(t, h, user), "PATCH", "/api/v1/admin/profile", `{"first_name":"  Priya ","last_name":"Nair   Menon"}`)
	expectStatus(t, recorder, 200)

	got := decodeProfile(t, recorder.Body.Bytes())
	if got.FirstName != "Priya" || got.LastName != "Nair Menon" || got.OrgName != h.tenant.OrgName {
		t.Fatalf("profile = %+v", got)
	}

	stored := storedUser(t, h, user.ID)
	if stored.FirstName != "Priya" || stored.LastName != "Nair Menon" {
		t.Fatalf("stored names = %q %q", stored.FirstName, stored.LastName)
	}
}

func TestProfileRenamesOrganizationWithPrivilege(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h, profilesvc.PrivilegeOrganizationEdit)

	recorder := do(t, profileRouter(t, h, user), "PATCH", "/api/v1/admin/profile", `{"org_name":"Acme   Privacy Labs"}`)
	expectStatus(t, recorder, 200)

	if got := decodeProfile(t, recorder.Body.Bytes()).OrgName; got != "Acme Privacy Labs" {
		t.Fatalf("org_name = %q", got)
	}
	if got := orgName(t, h, h.tenant.ID); got != "Acme Privacy Labs" {
		t.Fatalf("stored org_name = %q", got)
	}
	if got := orgName(t, h, h.other.ID); got != h.other.OrgName {
		t.Fatalf("another customer was renamed to %q", got)
	}
}

func TestProfileOrganizationRenameRequiresPrivilegeAndIsAtomic(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h)

	recorder := do(t, profileRouter(t, h, user), "PATCH", "/api/v1/admin/profile", `{"org_name":"Hijacked Org","first_name":"Changed"}`)
	expectStatus(t, recorder, 403)

	if got := decodeError(t, recorder).Error; got != utils.CodeForbidden {
		t.Fatalf("code = %s, want %s", got, utils.CodeForbidden)
	}
	if got := orgName(t, h, h.tenant.ID); got != h.tenant.OrgName {
		t.Fatalf("org renamed to %q without privilege", got)
	}
	if got := storedUser(t, h, user.ID).FirstName; got != "Asha" {
		t.Fatalf("first name changed to %q by a rejected request", got)
	}
}

func TestProfileUnchangedOrganizationNeedsNoPrivilege(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h)

	body := `{"org_name":"` + h.tenant.OrgName + `","first_name":"Meera"}`
	expectStatus(t, do(t, profileRouter(t, h, user), "PATCH", "/api/v1/admin/profile", body), 200)

	if got := storedUser(t, h, user.ID).FirstName; got != "Meera" {
		t.Fatalf("first name = %q", got)
	}
}

func TestProfileRejectsOrganizationNameTakenIgnoringCase(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h, profilesvc.PrivilegeOrganizationEdit)

	body := `{"org_name":"` + strings.ToUpper(h.other.OrgName) + `"}`
	recorder := do(t, profileRouter(t, h, user), "PATCH", "/api/v1/admin/profile", body)
	expectStatus(t, recorder, 409)

	if got := decodeError(t, recorder).Error; got != utils.CodeOrgNameTaken {
		t.Fatalf("code = %s, want %s", got, utils.CodeOrgNameTaken)
	}
	if got := orgName(t, h, h.tenant.ID); got != h.tenant.OrgName {
		t.Fatalf("org renamed to %q", got)
	}
}

func TestProfileRejectsInvalidInput(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h, profilesvc.PrivilegeOrganizationEdit)
	router := profileRouter(t, h, user)

	cases := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"admin email", `{"admin_email":"new@acme.test"}`, 400, utils.CodeAdminEmailImmutable},
		{"short org", `{"org_name":"A"}`, 400, utils.CodeValidation},
		{"blank first name", `{"first_name":"   "}`, 400, utils.CodeValidation},
		{"long last name", `{"last_name":"` + strings.Repeat("x", 51) + `"}`, 400, utils.CodeValidation},
		{"malformed", `{"first_name":`, 400, utils.CodeValidation},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := do(t, router, "PATCH", "/api/v1/admin/profile", testCase.body)
			expectStatus(t, recorder, testCase.status)

			if got := decodeError(t, recorder).Error; got != testCase.code {
				t.Fatalf("code = %s, want %s", got, testCase.code)
			}
		})
	}

	stored := storedUser(t, h, user.ID)
	if stored.FirstName != "Asha" || stored.LastName != "Rao" || stored.Email != user.Email {
		t.Fatalf("a rejected request changed the user: %+v", stored)
	}
}

func TestProfileOrganizationRenameDropsTenantIdentities(t *testing.T) {
	h := setup(t)
	user := profileUser(t, h, profilesvc.PrivilegeOrganizationEdit)

	store := auth.NewStore(h.rdb)
	ctx := context.Background()

	colleague := auth.IdentitySnapshot{User: auth.IdentityUser{ID: user.ID + 1000}, Customer: auth.IdentityCustomer{ID: h.tenant.ID}}
	outsider := auth.IdentitySnapshot{User: auth.IdentityUser{ID: user.ID + 2000}, Customer: auth.IdentityCustomer{ID: h.other.ID}}

	for _, snapshot := range []auth.IdentitySnapshot{colleague, outsider} {
		if err := store.CacheIdentity(ctx, snapshot.User.ID, snapshot, time.Hour); err != nil {
			t.Fatalf("cache identity failed: %v", err)
		}
	}

	expectStatus(t, do(t, profileRouter(t, h, user), "PATCH", "/api/v1/admin/profile", `{"org_name":"Renamed Org"}`), 200)

	if _, err := store.Identity(ctx, colleague.User.ID); !errors.Is(err, auth.ErrIdentityNotCached) {
		t.Fatalf("colleague identity still cached, err = %v", err)
	}
	if _, err := store.Identity(ctx, outsider.User.ID); err != nil {
		t.Fatalf("another customer's identity was dropped: %v", err)
	}
}
