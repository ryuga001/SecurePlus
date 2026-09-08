package deliveryaudit

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	adminutils "dpdp-backend/internal/admin/utils"
	dto "dpdp-backend/internal/audit/dto/deliveryaudit"
	service "dpdp-backend/internal/audit/services/deliveryaudit"
	"dpdp-backend/internal/audit/utils"
)

const PrivilegeAuditView = "admin.email.audit.view"

const dateLayout = "2006-01-02"

type DeliveryAuditHandler struct {
	svc *service.DeliveryAuditService
}

func NewDeliveryAuditHandler(svc *service.DeliveryAuditService) *DeliveryAuditHandler {
	return &DeliveryAuditHandler{svc: svc}
}

func (h *DeliveryAuditHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/audits")

	group.GET("", guard(PrivilegeAuditView), h.list)
	group.GET("/:correlationId", guard(PrivilegeAuditView), h.get)
}

func (h *DeliveryAuditHandler) list(c *gin.Context) {
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
		Status:   query.Status,
		Failure:  query.Failure,
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
	for _, audit := range listing.Items {
		items = append(items, toListItem(audit))
	}

	c.JSON(http.StatusOK, adminutils.ListResponse[dto.ListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *DeliveryAuditHandler) get(c *gin.Context) {
	actor, ok := adminutils.Actor(c)
	if !ok {
		return
	}

	audit, err := h.svc.Get(c.Request.Context(), actor.CustomerID, c.Param("correlationId"))
	if err != nil {
		h.respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(audit))
}

func (h *DeliveryAuditHandler) respond(c *gin.Context, err error) {
	if errors.Is(err, utils.ErrDeliveryAuditNotFound) {
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

func toListItem(audit dto.Audit) dto.ListItem {
	recipients := make([]string, 0, len(audit.Recipients))
	for _, recipient := range audit.Recipients {
		recipients = append(recipients, recipient.Email)
	}

	item := dto.ListItem{
		CorrelationID:  audit.CorrelationID,
		MessageID:      audit.MessageID,
		From:           audit.From,
		SenderDomain:   audit.SenderDomain,
		Recipients:     recipients,
		RecipientCount: len(recipients),
		Status:         audit.Status,
		AttemptCount:   len(audit.Attempts),
		Size:           audit.Size,
		CreatedAt:      audit.CreatedAt,
		UpdatedAt:      audit.UpdatedAt,
	}

	if audit.Failure != nil {
		item.FailureType = audit.Failure.Type
	}

	return item
}

func toResponse(audit dto.Audit) dto.Response {
	response := dto.Response{
		CorrelationID: audit.CorrelationID,
		MessageID:     audit.MessageID,
		ConfigID:      audit.ConfigID,
		From:          audit.From,
		SenderDomain:  audit.SenderDomain,
		Recipients:    make([]dto.RecipientResponse, 0, len(audit.Recipients)),
		Status:        audit.Status,
		DKIM: dto.DKIMResponse{
			Domain:   audit.DKIM.Domain,
			Selector: audit.DKIM.Selector,
			Signed:   audit.DKIM.Signed,
		},
		Attempts:  make([]dto.AttemptResponse, 0, len(audit.Attempts)),
		Size:      audit.Size,
		CreatedAt: audit.CreatedAt,
		UpdatedAt: audit.UpdatedAt,
	}

	for _, recipient := range audit.Recipients {
		response.Recipients = append(response.Recipients, dto.RecipientResponse{
			Email:    recipient.Email,
			Domain:   recipient.Domain,
			Status:   recipient.Status,
			SMTPCode: recipient.SMTPCode,
			Error:    recipient.Error,
		})
	}

	for _, attempt := range audit.Attempts {
		response.Attempts = append(response.Attempts, dto.AttemptResponse{
			Number:     attempt.Number,
			StartedAt:  attempt.StartedAt,
			FinishedAt: attempt.FinishedAt,
			MXHost:     attempt.MXHost,
			TLS:        attempt.TLS,
			SMTPCode:   attempt.SMTPCode,
			Error:      attempt.Error,
		})
	}

	if audit.Failure != nil {
		response.Failure = &dto.FailureResponse{
			Type:     audit.Failure.Type,
			Reason:   audit.Failure.Reason,
			SMTPCode: audit.Failure.SMTPCode,
		}
	}

	return response
}
