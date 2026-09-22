package alert

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/alert"
	repo "dpdp-backend/internal/admin/repositories/alert"
	service "dpdp-backend/internal/admin/services/alert"
	"dpdp-backend/internal/admin/utils"
)

const (
	PrivilegeAlertView   = "admin.email.alert.view"
	PrivilegeAlertCreate = "admin.email.alert.create"
	PrivilegeAlertEdit   = "admin.email.alert.edit"
	PrivilegeAlertDelete = "admin.email.alert.delete"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

func (h *AlertHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/alerts")

	group.GET("", guard(PrivilegeAlertView), h.list)
	group.GET("/:id", guard(PrivilegeAlertView), h.get)
	group.POST("", guard(PrivilegeAlertCreate), h.create)
	group.PUT("/:id", guard(PrivilegeAlertEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeAlertDelete), h.remove)
}

func (h *AlertHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query dto.AlertListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.ListParams{
		Search:           query.Search,
		NotificationType: query.NotificationType,
		ScheduleType:     query.ScheduleType,
		PolicyIDs:        service.ParsePolicyIDs(query.PolicyIDs),
		Page:             query.Page,
		PageSize:         query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.AlertListItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.AlertListItem{
			ID:               item.Alert.ID,
			Name:             item.Alert.Name,
			ScheduleType:     item.Alert.ScheduleType,
			NotificationType: item.Alert.NotificationType,
			AlertType:        item.Alert.AlertType,
			Policies:         toReferences(item.Policies),
			PolicyCount:      item.PolicyCount,
			TargetCount:      item.TargetCount,
			CreatedAt:        item.Alert.CreatedAt,
			UpdatedAt:        item.Alert.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.AlertListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *AlertHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndUUID(c)
	if !ok {
		return
	}

	detail, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toAlertResponse(detail))
}

func (h *AlertHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.AlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toAlertInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, toAlertResponse(detail))
}

func (h *AlertHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndUUID(c)
	if !ok {
		return
	}

	var req dto.AlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toAlertInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toAlertResponse(detail))
}

func (h *AlertHandler) remove(c *gin.Context) {
	actor, id, ok := utils.ActorAndUUID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), actor.CustomerID, id); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func toAlertInput(req dto.AlertRequest) service.AlertInput {
	return service.AlertInput{
		Name:             req.Name,
		ScheduleType:     req.ScheduleType,
		NotificationType: req.NotificationType,
		Target:           req.Target,
		PolicyIDs:        req.PolicyIDs,
	}
}

func toAlertResponse(detail service.AlertDetail) dto.AlertResponse {
	target := []string(detail.Alert.Target)
	if target == nil {
		target = []string{}
	}

	return dto.AlertResponse{
		ID:               detail.Alert.ID,
		Name:             detail.Alert.Name,
		ScheduleType:     detail.Alert.ScheduleType,
		NotificationType: detail.Alert.NotificationType,
		Target:           target,
		AlertType:        detail.Alert.AlertType,
		Policies:         toReferences(detail.Policies),
		CreatedAt:        detail.Alert.CreatedAt,
		UpdatedAt:        detail.Alert.UpdatedAt,
	}
}

func toReferences(rows []repo.AlertPolicyReference) []utils.ReferenceItem {
	items := make([]utils.ReferenceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, utils.ReferenceItem{ID: row.ID, Name: row.Name})
	}

	return items
}
