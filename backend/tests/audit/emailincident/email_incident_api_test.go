//go:build integration

package emailincident_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/audit/dto/emailincident"
	handler "dpdp-backend/internal/audit/handler/emailincident"
	repo "dpdp-backend/internal/audit/repositories/emailincident"
	service "dpdp-backend/internal/audit/services/emailincident"
	deliveryutils "dpdp-backend/internal/delivery/utils"
	"dpdp-backend/tests/testsupport"
)

func openGuard(_ string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func api(t *testing.T, customerID int) (*gin.Engine, *service.EmailIncidentService) {
	t.Helper()

	client := testsupport.Mongo(t)
	testsupport.ResetIncidents(t, client)

	t.Cleanup(func() { testsupport.ResetIncidents(t, client) })

	svc := service.NewEmailIncidentService(repo.NewEmailIncidentRepository(client, testsupport.MongoDatabase()))
	if err := svc.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("index creation failed: %v", err)
	}

	router, group := testsupport.Router(t, customerID, nil)
	handler.NewEmailIncidentHandler(svc).RegisterRoutes(group, openGuard)

	return router, svc
}

func seed(t *testing.T, svc *service.EmailIncidentService, customerID int, correlationID, trigger, action string) {
	t.Helper()

	record := dto.Record{
		CorrelationID:        correlationID,
		MessageID:            "<" + correlationID + "@sender.test>",
		CustomerID:           customerID,
		ConfigID:             1,
		From:                 "alice@example.com",
		SenderDomain:         "example.com",
		Recipients:           []dto.Recipient{{Email: "bob@recipient.test", Domain: "recipient.test"}},
		EmailUserID:          9,
		EvaluatedPolicyCount: 2,
		TriggeredPolicyIDs:   []int{1},
		Decision:             "FLAGGED",
		Trigger:              trigger,
		EffectiveAction:      action,
		ActionInvoked:        action,
		ActionStatus:         "PENDING",
		Matches: []dto.Match{{
			PolicyID:        1,
			PolicyName:      "Data Loss",
			RuleID:          5,
			RuleName:        "Cards",
			RuleType:        deliveryutils.MatcherKeyword,
			ConfiguredValue: "card",
			Occurrences:     2,
			Locations:       []string{deliveryutils.LocationBody},
		}},
	}

	if err := svc.Record(context.Background(), record); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
}

func get(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder
}

func TestListReturnsSeededIncidents(t *testing.T) {
	router, svc := api(t, 1)

	seed(t, svc, 1, "corr-a", "CONTENT", "QUARANTINE")
	seed(t, svc, 1, "corr-b", "RESTRICTION", "BLOCK")

	response := get(t, router, "/api/v1/admin/email/incidents")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var body struct {
		Items []dto.ListItem `json:"items"`
		Total int64          `json:"total"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if body.Total != 2 || len(body.Items) != 2 {
		t.Fatalf("total = %d, items = %d", body.Total, len(body.Items))
	}
	if body.Items[0].MatchCount != 1 || body.Items[0].RecipientCount != 1 {
		t.Fatalf("item = %+v", body.Items[0])
	}
}

func TestListFiltersByTrigger(t *testing.T) {
	router, svc := api(t, 1)

	seed(t, svc, 1, "corr-a", "CONTENT", "QUARANTINE")
	seed(t, svc, 1, "corr-b", "RESTRICTION", "BLOCK")

	response := get(t, router, "/api/v1/admin/email/incidents?trigger=RESTRICTION")

	var body struct {
		Items []dto.ListItem `json:"items"`
	}

	json.Unmarshal(response.Body.Bytes(), &body)

	if len(body.Items) != 1 || body.Items[0].Trigger != "RESTRICTION" {
		t.Fatalf("items = %+v", body.Items)
	}
}

func TestListRejectsUnknownTrigger(t *testing.T) {
	router, _ := api(t, 1)

	if response := get(t, router, "/api/v1/admin/email/incidents?trigger=NONSENSE"); response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestGetReturnsTheFullIncident(t *testing.T) {
	router, svc := api(t, 1)

	seed(t, svc, 1, "corr-a", "CONTENT", "QUARANTINE")

	response := get(t, router, "/api/v1/admin/email/incidents/corr-a")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var body dto.Response
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if body.CorrelationID != "corr-a" || len(body.Matches) != 1 {
		t.Fatalf("body = %+v", body)
	}
	if body.Matches[0].ConfiguredValue != "card" || body.Matches[0].Occurrences != 2 {
		t.Fatalf("match = %+v", body.Matches[0])
	}
}

func TestGetIsNotFoundForAnotherTenant(t *testing.T) {
	router, svc := api(t, 1)

	seed(t, svc, 2, "corr-other", "CONTENT", "QUARANTINE")

	response := get(t, router, "/api/v1/admin/email/incidents/corr-other")

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, cross-tenant access must be 404 not 403", response.Code)
	}
}

func TestListExcludesOtherTenants(t *testing.T) {
	router, svc := api(t, 1)

	seed(t, svc, 1, "corr-mine", "CONTENT", "QUARANTINE")
	seed(t, svc, 2, "corr-theirs", "CONTENT", "QUARANTINE")

	response := get(t, router, "/api/v1/admin/email/incidents")

	var body struct {
		Items []dto.ListItem `json:"items"`
		Total int64          `json:"total"`
	}

	json.Unmarshal(response.Body.Bytes(), &body)

	if body.Total != 1 || body.Items[0].CorrelationID != "corr-mine" {
		t.Fatalf("items = %+v, another tenant's incident leaked", body.Items)
	}
}

func TestGetIsNotFoundForUnknownCorrelationID(t *testing.T) {
	router, _ := api(t, 1)

	if response := get(t, router, "/api/v1/admin/email/incidents/missing"); response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
}
