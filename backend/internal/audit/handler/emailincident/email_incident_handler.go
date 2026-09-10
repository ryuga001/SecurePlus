package emailincident

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	adminutils "dpdp-backend/internal/admin/utils"
	dto "dpdp-backend/internal/audit/dto/emailincident"
	service "dpdp-backend/internal/audit/services/emailincident"
	"dpdp-backend/internal/audit/utils"
)

const PrivilegeIncidentView = "admin.email.incident.view"

const dateLayout = "2006-01-02"

type EmailIncidentHandler struct {
	svc *service.EmailIncidentService
}

func NewEmailIncidentHandler(svc *service.EmailIncidentService) *EmailIncidentHandler {
	return &EmailIncidentHandler{svc: svc}
}

func (h *EmailIncidentHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/incidents")

	group.GET("", guard(PrivilegeIncidentView), h.list)
	group.GET("/:correlationId", guard(PrivilegeIncidentView), h.get)
}

func (h *EmailIncidentHandler) list(c *gin.Context) {
	actor, ok := adminutils.Actor(c)
	if !ok {
		return
	}

	var query dto.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		adminutils.BadRequest(c)
		return
	}

	from, ok := parseBoundary(c, query.CreatedFrom, false)
	if !ok {
		return
	}

	to, ok := parseBoundary(c, query.CreatedTo, true)
	if !ok {
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, dto.ListParams{
		Search:   query.Search,
		Trigger:  query.Trigger,
		Action:   query.Action,
		From:     from,
		To:       to,
		SortBy:   query.SortBy,
		SortDesc: query.SortDir != "asc",
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		h.respond(c, err)
		return
	}

	items := make([]dto.ListItem, 0, len(listing.Items))
	for _, incident := range listing.Items {
		items = append(items, toListItem(incident))
	}

	c.JSON(http.StatusOK, adminutils.ListResponse[dto.ListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *EmailIncidentHandler) get(c *gin.Context) {
	actor, ok := adminutils.Actor(c)
	if !ok {
		return
	}

	incident, err := h.svc.Get(c.Request.Context(), actor.CustomerID, c.Param("correlationId"))
	if err != nil {
		h.respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(incident))
}

func (h *EmailIncidentHandler) respond(c *gin.Context, err error) {
	if errors.Is(err, utils.ErrEmailIncidentNotFound) {
		adminutils.Fail(c, http.StatusNotFound, utils.CodeNotFound, err.Error())
		return
	}

	adminutils.Respond(c, err)
}

func parseBoundary(c *gin.Context, value string, endOfDay bool) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, true
	}

	if moment, err := time.Parse(time.RFC3339, value); err == nil {
		return moment.UTC(), true
	}

	day, err := time.Parse(dateLayout, value)
	if err != nil {
		adminutils.BadRequest(c)
		return time.Time{}, false
	}

	if endOfDay {
		return day.UTC().Add(24*time.Hour - time.Nanosecond), true
	}

	return day.UTC(), true
}

func toListItem(incident dto.Incident) dto.ListItem {
	recipients := make([]string, 0, len(incident.Recipients))
	for _, recipient := range incident.Recipients {
		recipients = append(recipients, recipient.Email)
	}

	return dto.ListItem{
		CorrelationID:   incident.CorrelationID,
		MessageID:       incident.MessageID,
		From:            incident.From,
		SenderDomain:    incident.SenderDomain,
		Recipients:      recipients,
		RecipientCount:  len(recipients),
		Decision:        incident.Decision,
		Trigger:         incident.Trigger,
		EffectiveAction: incident.EffectiveAction,
		ActionStatus:    incident.ActionStatus,
		WithheldCount:   len(incident.WithheldRecipients),
		ViolationCount:  len(incident.RestrictionViolations),
		MatchCount:      len(incident.Matches),
		CreatedAt:       incident.CreatedAt,
	}
}

func toResponse(incident dto.Incident) dto.Response {
	response := dto.Response{
		CorrelationID:         incident.CorrelationID,
		MessageID:             incident.MessageID,
		ConfigID:              incident.ConfigID,
		From:                  incident.From,
		SenderDomain:          incident.SenderDomain,
		Recipients:            make([]dto.RecipientResponse, 0, len(incident.Recipients)),
		EmailUserID:           incident.EmailUserID,
		EvaluatedPolicyCount:  incident.EvaluatedPolicyCount,
		TriggeredPolicyIDs:    incident.TriggeredPolicyIDs,
		Decision:              incident.Decision,
		Trigger:               incident.Trigger,
		EffectiveAction:       incident.EffectiveAction,
		ActionInvoked:         incident.ActionInvoked,
		ActionStatus:          incident.ActionStatus,
		ActionError:           incident.ActionError,
		WithheldRecipients:    make([]dto.WithheldResponse, 0, len(incident.WithheldRecipients)),
		RestrictionViolations: make([]dto.ViolationResponse, 0, len(incident.RestrictionViolations)),
		Matches:               make([]dto.MatchResponse, 0, len(incident.Matches)),
		CreatedAt:             incident.CreatedAt,
		UpdatedAt:             incident.UpdatedAt,
	}

	for _, recipient := range incident.Recipients {
		response.Recipients = append(response.Recipients, dto.RecipientResponse{
			Email:  recipient.Email,
			Domain: recipient.Domain,
		})
	}

	for _, recipient := range incident.WithheldRecipients {
		response.WithheldRecipients = append(response.WithheldRecipients, dto.WithheldResponse{
			Email:      recipient.Email,
			Domain:     recipient.Domain,
			PolicyID:   recipient.PolicyID,
			PolicyName: recipient.PolicyName,
		})
	}

	for _, violation := range incident.RestrictionViolations {
		response.RestrictionViolations = append(response.RestrictionViolations, dto.ViolationResponse{
			Kind:        violation.Kind,
			Mode:        violation.Mode,
			Value:       violation.Value,
			Filename:    violation.Filename,
			ContentType: violation.ContentType,
			PolicyID:    violation.PolicyID,
			PolicyName:  violation.PolicyName,
		})
	}

	for _, match := range incident.Matches {
		response.Matches = append(response.Matches, dto.MatchResponse{
			PolicyID:        match.PolicyID,
			PolicyName:      match.PolicyName,
			RuleID:          match.RuleID,
			RuleName:        match.RuleName,
			RuleType:        match.RuleType,
			ConfiguredValue: match.ConfiguredValue,
			Occurrences:     match.Occurrences,
			Locations:       match.Locations,
		})
	}

	return response
}
