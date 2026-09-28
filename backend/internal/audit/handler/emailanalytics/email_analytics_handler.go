package emailanalytics

import (
	"net/http"

	"github.com/gin-gonic/gin"

	adminutils "dpdp-backend/internal/admin/utils"
	dto "dpdp-backend/internal/audit/dto/emailanalytics"
	service "dpdp-backend/internal/audit/services/emailanalytics"
)

const PrivilegeIncidentView = "admin.email.incident.view"

type EmailAnalyticsHandler struct {
	svc *service.EmailAnalyticsService
}

func NewEmailAnalyticsHandler(svc *service.EmailAnalyticsService) *EmailAnalyticsHandler {
	return &EmailAnalyticsHandler{svc: svc}
}

func (h *EmailAnalyticsHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/analytics")

	group.GET("", guard(PrivilegeIncidentView), h.analytics)
}

func (h *EmailAnalyticsHandler) analytics(c *gin.Context) {
	actor, ok := adminutils.Actor(c)
	if !ok {
		return
	}

	var query dto.AnalyticsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		adminutils.BadRequest(c)
		return
	}

	analytics, err := h.svc.Analytics(c.Request.Context(), actor.CustomerID, query.Period)
	if err != nil {
		adminutils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(analytics))
}

func toResponse(analytics dto.Analytics) dto.AnalyticsResponse {
	response := dto.AnalyticsResponse{
		Period:      analytics.Period,
		TrendWindow: toWindow(analytics.TrendWindow),
		TopWindow:   toWindow(analytics.TopWindow),
		Summary: dto.SummaryResponse{
			Total:   analytics.Summary.Total,
			Blocked: analytics.Summary.Blocked,
			Flagged: analytics.Summary.Flagged,
		},
		Trend: dto.TrendResponse{
			Bucket: analytics.Bucket,
			Points: make([]dto.TrendPointResponse, 0, len(analytics.Trend)),
		},
		TopUsers:    make([]dto.UserRankingResponse, 0, len(analytics.TopUsers)),
		TopPolicies: make([]dto.PolicyRankingResponse, 0, len(analytics.TopPolicies)),
		TopRules:    make([]dto.RuleRankingResponse, 0, len(analytics.TopRules)),
	}

	for _, point := range analytics.Trend {
		response.Trend.Points = append(response.Trend.Points, dto.TrendPointResponse{
			BucketStart: point.BucketStart,
			Total:       point.Total,
			Blocked:     point.Blocked,
			Flagged:     point.Flagged,
		})
	}

	for _, row := range analytics.TopUsers {
		response.TopUsers = append(response.TopUsers, dto.UserRankingResponse{
			Email:   row.Email,
			Total:   row.Total,
			Blocked: row.Blocked,
			Flagged: row.Flagged,
		})
	}

	for _, row := range analytics.TopPolicies {
		response.TopPolicies = append(response.TopPolicies, dto.PolicyRankingResponse{
			PolicyID:   row.PolicyID,
			PolicyName: row.PolicyName,
			Count:      row.Count,
		})
	}

	for _, row := range analytics.TopRules {
		response.TopRules = append(response.TopRules, dto.RuleRankingResponse{
			RuleID:   row.RuleID,
			RuleName: row.RuleName,
			RuleType: row.RuleType,
			Count:    row.Count,
		})
	}

	return response
}

func toWindow(window dto.Window) dto.WindowResponse {
	return dto.WindowResponse{From: window.From, To: window.To}
}
