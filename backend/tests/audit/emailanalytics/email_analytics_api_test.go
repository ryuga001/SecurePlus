//go:build integration

package emailanalytics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	dto "dpdp-backend/internal/audit/dto/emailanalytics"
	handler "dpdp-backend/internal/audit/handler/emailanalytics"
	repo "dpdp-backend/internal/audit/repositories/emailanalytics"
	service "dpdp-backend/internal/audit/services/emailanalytics"
	"dpdp-backend/internal/audit/utils"
	deliveryutils "dpdp-backend/internal/delivery/utils"
	"dpdp-backend/tests/testsupport"
)

const tenant = 4100

func openGuard(_ string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func api(t *testing.T, customerID int) (*gin.Engine, *mongo.Client) {
	t.Helper()

	client := testsupport.Mongo(t)
	testsupport.ResetIncidents(t, client)

	t.Cleanup(func() { testsupport.ResetIncidents(t, client) })

	svc := service.NewEmailAnalyticsService(repo.NewEmailAnalyticsRepository(client, testsupport.MongoDatabase()))

	router, group := testsupport.Router(t, customerID, nil)
	handler.NewEmailAnalyticsHandler(svc).RegisterRoutes(group, openGuard)

	return router, client
}

func match(policyID int, policyName string, ruleID int, ruleName, ruleType string) bson.M {
	return bson.M{
		"policy_id":   policyID,
		"policy_name": policyName,
		"rule_id":     ruleID,
		"rule_name":   ruleName,
		"rule_type":   ruleType,
		"occurrences": 1,
		"locations":   []string{"body"},
	}
}

func seed(t *testing.T, client *mongo.Client, customerID int, correlationID, from, action string, at time.Time, matches ...bson.M) {
	t.Helper()

	if matches == nil {
		matches = []bson.M{}
	}

	document := bson.M{
		"correlation_id":         correlationID,
		"message_id":             "<" + correlationID + "@sender.test>",
		"customer_id":            customerID,
		"config_id":              1,
		"from":                   from,
		"sender_domain":          "acme.test",
		"recipients":             []bson.M{{"email": "out@partner.test", "domain": "partner.test"}},
		"email_user_id":          7,
		"evaluated_policy_count": 2,
		"triggered_policy_ids":   []int{1},
		"decision":               "FLAGGED",
		"trigger":                "CONTENT",
		"effective_action":       action,
		"action_invoked":         action,
		"action_status":          "INVOKED",
		"matches":                matches,
		"created_at":             at.UTC(),
		"updated_at":             at.UTC(),
	}

	collection := client.Database(testsupport.MongoDatabase()).Collection(utils.EmailIncidentCollection)

	if _, err := collection.InsertOne(context.Background(), document); err != nil {
		t.Fatalf("seed %s failed: %v", correlationID, err)
	}
}

func fetch(t *testing.T, router *gin.Engine, period string) dto.AnalyticsResponse {
	t.Helper()

	url := "/api/v1/admin/email/analytics"
	if period != "" {
		url += "?period=" + period
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, url, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", recorder.Code, recorder.Body.String())
	}

	var payload dto.AnalyticsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not json: %v (%s)", err, recorder.Body.String())
	}

	return payload
}

func TestAnalyticsBucketsDailyTrendAndZeroFills(t *testing.T) {
	router, client := api(t, tenant)
	now := time.Now().UTC()

	seed(t, client, tenant, "d-today-1", "asha@acme.test", deliveryutils.ActionBlock, now.Add(-2*time.Hour))
	seed(t, client, tenant, "d-today-2", "asha@acme.test", deliveryutils.ActionAudit, now.Add(-3*time.Hour))
	seed(t, client, tenant, "d-back-3", "vikram@acme.test", deliveryutils.ActionAudit, now.AddDate(0, 0, -3).Add(-time.Hour))

	payload := fetch(t, router, dto.PeriodDay)

	if payload.Period != dto.PeriodDay || payload.Trend.Bucket != dto.PeriodDay {
		t.Fatalf("period = %s bucket = %s", payload.Period, payload.Trend.Bucket)
	}
	if len(payload.Trend.Points) != 14 {
		t.Fatalf("points = %d, want 14", len(payload.Trend.Points))
	}

	last := payload.Trend.Points[13]
	if last.Total != 2 || last.Blocked != 1 || last.Flagged != 1 {
		t.Fatalf("today bucket = %+v", last)
	}

	if older := payload.Trend.Points[10]; older.Total != 1 || older.Blocked != 0 || older.Flagged != 1 {
		t.Fatalf("three-days-ago bucket = %+v", older)
	}

	var empty int
	for _, point := range payload.Trend.Points {
		if point.Total == 0 {
			empty++
		}
	}

	if empty != 12 {
		t.Fatalf("zero-filled buckets = %d, want 12", empty)
	}
}

func TestAnalyticsSummaryCountsTheTopWindowOnly(t *testing.T) {
	router, client := api(t, tenant)
	now := time.Now().UTC()

	seed(t, client, tenant, "s-recent-1", "asha@acme.test", deliveryutils.ActionBlock, now.Add(-2*time.Hour))
	seed(t, client, tenant, "s-recent-2", "asha@acme.test", deliveryutils.ActionRedact, now.Add(-4*time.Hour))
	seed(t, client, tenant, "s-old", "asha@acme.test", deliveryutils.ActionBlock, now.AddDate(0, 0, -5))

	payload := fetch(t, router, dto.PeriodDay)

	if payload.Summary.Total != 2 || payload.Summary.Blocked != 1 || payload.Summary.Flagged != 1 {
		t.Fatalf("summary = %+v, want only the last 24h", payload.Summary)
	}

	var trend int64
	for _, point := range payload.Trend.Points {
		trend += point.Total
	}

	if trend != 3 {
		t.Fatalf("trend total = %d, want all 3 incidents in the 14-day window", trend)
	}
}

func TestAnalyticsRanksTopSendersWithSplit(t *testing.T) {
	router, client := api(t, tenant)
	now := time.Now().UTC()

	senders := map[string]int{"top@acme.test": 4, "mid@acme.test": 2, "low@acme.test": 1}

	for sender, count := range senders {
		for index := range count {
			action := deliveryutils.ActionAudit
			if index == 0 {
				action = deliveryutils.ActionBlock
			}

			seed(t, client, tenant, fmt.Sprintf("u-%s-%d", sender, index), sender, action, now.Add(-time.Duration(index+1)*time.Hour))
		}
	}

	payload := fetch(t, router, dto.PeriodDay)

	if len(payload.TopUsers) != 3 {
		t.Fatalf("top users = %+v", payload.TopUsers)
	}

	first := payload.TopUsers[0]
	if first.Email != "top@acme.test" || first.Total != 4 || first.Blocked != 1 || first.Flagged != 3 {
		t.Fatalf("first = %+v", first)
	}

	for index := 1; index < len(payload.TopUsers); index++ {
		if payload.TopUsers[index].Total > payload.TopUsers[index-1].Total {
			t.Fatalf("top users are not descending: %+v", payload.TopUsers)
		}
	}
}

func TestAnalyticsCapsRankingsAtFive(t *testing.T) {
	router, client := api(t, tenant)
	now := time.Now().UTC()

	for index := range 7 {
		seed(t, client, tenant,
			fmt.Sprintf("cap-%d", index),
			fmt.Sprintf("sender%d@acme.test", index),
			deliveryutils.ActionAudit,
			now.Add(-time.Duration(index+1)*time.Minute),
			match(index+1, fmt.Sprintf("Policy %d", index), index+1, fmt.Sprintf("Rule %d", index), "REGEX"),
		)
	}

	payload := fetch(t, router, dto.PeriodDay)

	if len(payload.TopUsers) != 5 || len(payload.TopPolicies) != 5 || len(payload.TopRules) != 5 {
		t.Fatalf("caps: users %d policies %d rules %d", len(payload.TopUsers), len(payload.TopPolicies), len(payload.TopRules))
	}
}

func TestAnalyticsCountsPoliciesAndRulesOncePerIncident(t *testing.T) {
	router, client := api(t, tenant)
	now := time.Now().UTC()

	seed(t, client, tenant, "dedupe-1", "asha@acme.test", deliveryutils.ActionBlock, now.Add(-time.Hour),
		match(9, "PAN in attachments", 21, "PAN regex", "REGEX"),
		match(9, "PAN in attachments", 22, "PAN keyword", "KEYWORD"),
		match(9, "PAN in attachments", 23, "Card regex", "REGEX"),
	)
	seed(t, client, tenant, "dedupe-2", "vikram@acme.test", deliveryutils.ActionAudit, now.Add(-2*time.Hour),
		match(9, "PAN in attachments", 21, "PAN regex", "REGEX"),
	)

	payload := fetch(t, router, dto.PeriodDay)

	if len(payload.TopPolicies) != 1 {
		t.Fatalf("policies = %+v", payload.TopPolicies)
	}

	if policy := payload.TopPolicies[0]; policy.PolicyID != 9 || policy.Count != 2 {
		t.Fatalf("policy = %+v, want 2 incidents not 4 matches", policy)
	}

	counts := make(map[string]int64, len(payload.TopRules))
	for _, rule := range payload.TopRules {
		counts[rule.RuleName] = rule.Count
	}

	if counts["PAN regex"] != 2 || counts["PAN keyword"] != 1 || counts["Card regex"] != 1 {
		t.Fatalf("rule counts = %v", counts)
	}

	if payload.TopRules[0].RuleName != "PAN regex" || payload.TopRules[0].RuleType != "REGEX" {
		t.Fatalf("top rule = %+v", payload.TopRules[0])
	}
}

func TestAnalyticsIsTenantScoped(t *testing.T) {
	router, client := api(t, tenant)
	now := time.Now().UTC()

	seed(t, client, tenant, "mine", "asha@acme.test", deliveryutils.ActionBlock, now.Add(-time.Hour),
		match(1, "Mine", 1, "Mine rule", "REGEX"))
	seed(t, client, tenant+1, "theirs", "outsider@other.test", deliveryutils.ActionBlock, now.Add(-time.Hour),
		match(2, "Theirs", 2, "Theirs rule", "REGEX"))

	payload := fetch(t, router, dto.PeriodDay)

	if payload.Summary.Total != 1 {
		t.Fatalf("summary = %+v, want only this tenant's incident", payload.Summary)
	}
	if len(payload.TopUsers) != 1 || payload.TopUsers[0].Email != "asha@acme.test" {
		t.Fatalf("top users leaked: %+v", payload.TopUsers)
	}
	if len(payload.TopPolicies) != 1 || payload.TopPolicies[0].PolicyName != "Mine" {
		t.Fatalf("top policies leaked: %+v", payload.TopPolicies)
	}
	if len(payload.TopRules) != 1 || payload.TopRules[0].RuleName != "Mine rule" {
		t.Fatalf("top rules leaked: %+v", payload.TopRules)
	}
}

func TestAnalyticsServesEveryPeriodAndDefaultsToWeek(t *testing.T) {
	router, _ := api(t, tenant)

	for period, want := range map[string]int{dto.PeriodDay: 14, dto.PeriodWeek: 8, dto.PeriodMonth: 6} {
		payload := fetch(t, router, period)

		if payload.Period != period || payload.Trend.Bucket != period {
			t.Fatalf("period %s → %+v", period, payload.Period)
		}
		if len(payload.Trend.Points) != want {
			t.Fatalf("period %s: points = %d, want %d", period, len(payload.Trend.Points), want)
		}
		if payload.TopUsers == nil || payload.TopPolicies == nil || payload.TopRules == nil {
			t.Fatalf("period %s: rankings must be empty arrays, never null", period)
		}
	}

	if payload := fetch(t, router, ""); payload.Period != dto.PeriodWeek || len(payload.Trend.Points) != 8 {
		t.Fatalf("default = %s with %d points", payload.Period, len(payload.Trend.Points))
	}
}

func TestAnalyticsRejectsUnknownPeriod(t *testing.T) {
	router, _ := api(t, tenant)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/email/analytics?period=quarter", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}
