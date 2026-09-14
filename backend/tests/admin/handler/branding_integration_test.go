//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	brandinghandler "dpdp-backend/internal/admin/handler/branding"
	brandingsvc "dpdp-backend/internal/admin/services/branding"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/db"
	"dpdp-backend/tests/testsupport"
)

func brandingRow(t *testing.T, h harness, customerID int) db.CustomerBranding {
	t.Helper()

	var row db.CustomerBranding
	if err := h.database.Where("customer_id = ?", customerID).Take(&row).Error; err != nil {
		t.Fatalf("branding lookup failed: %v", err)
	}

	return row
}

func brandingRouter(t *testing.T, h harness, customerID int) *gin.Engine {
	t.Helper()

	router, group := testsupport.Router(t, customerID, nil)

	pass := func(string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}

	brandinghandler.NewBrandingHandler(h.branding).RegisterRoutes(group, pass)

	return router
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	canvas.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		t.Fatalf("png encode failed: %v", err)
	}

	return buffer.Bytes()
}

func multipartBody(t *testing.T, filename string, payload []byte) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("logo", filename)
	if err != nil {
		t.Fatalf("multipart part failed: %v", err)
	}

	if _, err := part.Write(payload); err != nil {
		t.Fatalf("multipart write failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close failed: %v", err)
	}

	return &body, writer.FormDataContentType()
}

func doUpload(t *testing.T, router *gin.Engine, filename string, payload []byte) *httptest.ResponseRecorder {
	t.Helper()

	body, contentType := multipartBody(t, filename, payload)

	request := httptest.NewRequest("POST", "/api/v1/admin/branding/logo", body)
	request.Header.Set("Content-Type", contentType)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func TestBrandingDefaultsExistForEveryCustomer(t *testing.T) {
	h := setup(t)

	for _, customer := range []db.Customer{h.tenant, h.other} {
		row := brandingRow(t, h, customer.ID)

		if row.Theme != db.ThemeLight || row.Language != db.LanguageEnglish || row.Timezone != db.TimezoneUTC {
			t.Fatalf("defaults = %s/%s/%s, want LIGHT/ENGLISH/UTC", row.Theme, row.Language, row.Timezone)
		}

		if row.LogoKey != nil {
			t.Fatalf("logo_key = %v, want nil", *row.LogoKey)
		}
	}
}

func TestBrandingPartialUpdateLeavesOtherFields(t *testing.T) {
	h := setup(t)
	router := brandingRouter(t, h, h.tenant.ID)

	expectStatus(t, do(t, router, "PATCH", "/api/v1/admin/branding", `{"theme":"DARK"}`), 204)

	row := brandingRow(t, h, h.tenant.ID)
	if row.Theme != db.ThemeDark {
		t.Fatalf("theme = %s, want DARK", row.Theme)
	}
	if row.Language != db.LanguageEnglish || row.Timezone != db.TimezoneUTC {
		t.Fatalf("partial update changed %s/%s", row.Language, row.Timezone)
	}

	expectStatus(t, do(t, router, "PATCH", "/api/v1/admin/branding", `{"timezone":"Asia/Tokyo"}`), 204)

	row = brandingRow(t, h, h.tenant.ID)
	if row.Timezone != "Asia/Tokyo" {
		t.Fatalf("timezone = %s, want Asia/Tokyo", row.Timezone)
	}
	if row.Theme != db.ThemeDark || row.Language != db.LanguageEnglish {
		t.Fatalf("timezone update changed %s/%s", row.Theme, row.Language)
	}
}

func TestBrandingRejectsInvalidValues(t *testing.T) {
	h := setup(t)
	router := brandingRouter(t, h, h.tenant.ID)

	cases := []struct {
		name string
		body string
		code string
	}{
		{"theme", `{"theme":"BLUE"}`, utils.CodeInvalidTheme},
		{"language", `{"language":"FRENCH"}`, utils.CodeInvalidLanguage},
		{"timezone", `{"timezone":"IST"}`, utils.CodeInvalidTimezone},
		{"offset timezone", `{"timezone":"+05:30"}`, utils.CodeInvalidTimezone},
		{"empty timezone", `{"timezone":" "}`, utils.CodeInvalidTimezone},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := do(t, router, "PATCH", "/api/v1/admin/branding", testCase.body)
			expectStatus(t, recorder, 400)

			if got := decodeError(t, recorder).Error; got != testCase.code {
				t.Fatalf("code = %s, want %s", got, testCase.code)
			}
		})
	}

	row := brandingRow(t, h, h.tenant.ID)
	if row.Theme != db.ThemeLight || row.Language != db.LanguageEnglish || row.Timezone != db.TimezoneUTC {
		t.Fatal("a rejected request changed stored branding")
	}
}

func TestBrandingUpdateCannotReachAnotherCustomer(t *testing.T) {
	h := setup(t)
	router := brandingRouter(t, h, h.tenant.ID)

	expectStatus(t, do(t, router, "PATCH", "/api/v1/admin/branding", `{"theme":"DARK"}`), 204)

	other := brandingRow(t, h, h.other.ID)
	if other.Theme != db.ThemeLight {
		t.Fatalf("customer B theme = %s, want LIGHT", other.Theme)
	}
}

func TestBrandingUpdateDropsEveryIdentityOfThatCustomerOnly(t *testing.T) {
	h := setup(t)

	store := auth.NewStore(h.rdb)
	ctx := context.Background()

	tenantOne := auth.IdentitySnapshot{
		User:     auth.IdentityUser{ID: 11},
		Customer: auth.IdentityCustomer{ID: h.tenant.ID},
	}
	tenantTwo := auth.IdentitySnapshot{
		User:     auth.IdentityUser{ID: 12},
		Customer: auth.IdentityCustomer{ID: h.tenant.ID},
	}
	outsider := auth.IdentitySnapshot{
		User:     auth.IdentityUser{ID: 21},
		Customer: auth.IdentityCustomer{ID: h.other.ID},
	}

	for _, snapshot := range []auth.IdentitySnapshot{tenantOne, tenantTwo, outsider} {
		if err := store.CacheIdentity(ctx, snapshot.User.ID, snapshot, time.Hour); err != nil {
			t.Fatalf("cache seed failed: %v", err)
		}
	}

	router := brandingRouter(t, h, h.tenant.ID)
	expectStatus(t, do(t, router, "PATCH", "/api/v1/admin/branding", `{"theme":"DARK"}`), 204)

	for _, userID := range []int{tenantOne.User.ID, tenantTwo.User.ID} {
		if _, err := store.Identity(ctx, userID); !errors.Is(err, auth.ErrIdentityNotCached) {
			t.Fatalf("identity %d still cached after branding update (err = %v)", userID, err)
		}
	}

	if _, err := store.Identity(ctx, outsider.User.ID); err != nil {
		t.Fatalf("other customer identity was evicted: %v", err)
	}
}

func TestBrandingSnapshotServesStoredPreferences(t *testing.T) {
	h := setup(t)
	router := brandingRouter(t, h, h.tenant.ID)

	expectStatus(t, do(t, router, "PATCH", "/api/v1/admin/branding",
		`{"theme":"DARK","language":"JAPANESE","timezone":"Asia/Tokyo"}`), 204)

	snapshot, err := h.branding.Snapshot(context.Background(), h.tenant.ID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if snapshot.Theme != db.ThemeDark || snapshot.Language != db.LanguageJapanese || snapshot.Timezone != "Asia/Tokyo" {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	if snapshot.LogoURL != nil {
		t.Fatalf("logo_url = %v, want nil", *snapshot.LogoURL)
	}
}

func TestBrandingLogoRejectsBadUploads(t *testing.T) {
	h := setup(t)
	router := brandingRouter(t, h, h.tenant.ID)

	oversized := make([]byte, brandingsvc.MaxLogoSize+1)
	copy(oversized, pngBytes(t, 8, 8))

	cases := []struct {
		name     string
		filename string
		payload  []byte
	}{
		{"oversized", "logo.png", oversized},
		{"not an image", "logo.png", []byte("this is plain text pretending to be a png")},
		{"empty", "logo.png", nil},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := doUpload(t, router, testCase.filename, testCase.payload)
			expectStatus(t, recorder, 400)

			if got := decodeError(t, recorder).Error; got != utils.CodeInvalidLogo {
				t.Fatalf("code = %s, want %s", got, utils.CodeInvalidLogo)
			}
		})
	}

	if row := brandingRow(t, h, h.tenant.ID); row.LogoKey != nil {
		t.Fatalf("logo_key = %v after failed uploads, want nil", *row.LogoKey)
	}
}
