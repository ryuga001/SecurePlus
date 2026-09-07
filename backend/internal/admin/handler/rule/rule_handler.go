package rule

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/rule"
	repo "dpdp-backend/internal/admin/repositories/rule"
	service "dpdp-backend/internal/admin/services/rule"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	PrivilegeRuleView   = "admin.rule.view"
	PrivilegeRuleCreate = "admin.rule.create"
	PrivilegeRuleEdit   = "admin.rule.edit"
	PrivilegeRuleDelete = "admin.rule.delete"
)

type RuleHandler struct {
	svc *service.RuleService
}

func NewRuleHandler(svc *service.RuleService) *RuleHandler {
	return &RuleHandler{svc: svc}
}

func (h *RuleHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/rules")

	group.GET("", guard(PrivilegeRuleView), h.list)
	group.GET("/:id", guard(PrivilegeRuleView), h.get)
	group.POST("", guard(PrivilegeRuleCreate), h.create)
	group.PUT("/:id", guard(PrivilegeRuleEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeRuleDelete), h.remove)
}

func (h *RuleHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query utils.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.ListParams{
		Search:   query.Search,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.RuleResponse, 0, len(listing.Items))
	for _, row := range listing.Items {
		items = append(items, toRuleResponse(row))
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.RuleResponse]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *RuleHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toRuleResponse(row))
}

func (h *RuleHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toRuleInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, toRuleResponse(row))
}

func (h *RuleHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toRuleInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toRuleResponse(row))
}

func (h *RuleHandler) remove(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), actor.CustomerID, id); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func toRuleInput(req dto.RuleRequest) service.RuleInput {
	return service.RuleInput{RuleName: req.RuleName, Type: req.Type, Value: req.Value}
}

func toRuleResponse(row db.Rule) dto.RuleResponse {
	return dto.RuleResponse{
		ID:        row.ID,
		RuleName:  row.RuleName,
		Type:      row.Type,
		Value:     row.Value,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
