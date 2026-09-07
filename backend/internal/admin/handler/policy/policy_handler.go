package policy

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/policy"
	repo "dpdp-backend/internal/admin/repositories/policy"
	service "dpdp-backend/internal/admin/services/policy"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	PrivilegePolicyView   = "admin.policy.view"
	PrivilegePolicyCreate = "admin.policy.create"
	PrivilegePolicyEdit   = "admin.policy.edit"
	PrivilegePolicyDelete = "admin.policy.delete"
)

type PolicyHandler struct {
	svc *service.PolicyService
}

func NewPolicyHandler(svc *service.PolicyService) *PolicyHandler {
	return &PolicyHandler{svc: svc}
}

func (h *PolicyHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/policies")

	group.GET("", guard(PrivilegePolicyView), h.list)
	group.GET("/:id", guard(PrivilegePolicyView), h.get)
	group.POST("", guard(PrivilegePolicyCreate), h.create)
	group.PUT("/:id", guard(PrivilegePolicyEdit), h.update)
	group.PATCH("/:id/status", guard(PrivilegePolicyEdit), h.setStatus)
	group.DELETE("/:id", guard(PrivilegePolicyDelete), h.remove)
}

func (h *PolicyHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query dto.PolicyListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.ListParams{
		Search:   query.Search,
		Active:   query.Active,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.PolicyListItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.PolicyListItem{
			ID:         item.Policy.ID,
			PolicyName: item.Policy.PolicyName,
			Type:       item.Policy.Type,
			Active:     item.Policy.Active,
			GroupCount: item.GroupCount,
			RuleCount:  item.RuleCount,
			CreatedAt:  item.Policy.CreatedAt,
			UpdatedAt:  item.Policy.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.PolicyListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *PolicyHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	detail, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toPolicyResponse(detail))
}

func (h *PolicyHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toPolicyInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, toPolicyResponse(detail))
}

func (h *PolicyHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toPolicyInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toPolicyResponse(detail))
}

func (h *PolicyHandler) setStatus(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.PolicyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.SetActive(c.Request.Context(), actor.CustomerID, id, *req.Active)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toPolicyResponse(detail))
}

func (h *PolicyHandler) remove(c *gin.Context) {
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

func toPolicyInput(req dto.PolicyRequest) service.PolicyInput {
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	return service.PolicyInput{
		PolicyName: req.PolicyName,
		Active:     active,
		GroupIDs:   req.GroupIDs,
		RuleIDs:    req.RuleIDs,
	}
}

func toPolicyResponse(detail service.PolicyDetail) dto.PolicyResponse {
	return dto.PolicyResponse{
		ID:         detail.Policy.ID,
		PolicyName: detail.Policy.PolicyName,
		Type:       db.PolicyTypeEmail,
		Active:     detail.Policy.Active,
		Groups:     toReferences(detail.Groups),
		Rules:      toReferences(detail.Rules),
		CreatedAt:  detail.Policy.CreatedAt,
		UpdatedAt:  detail.Policy.UpdatedAt,
	}
}

func toReferences(rows []repo.PolicyReference) []utils.ReferenceItem {
	items := make([]utils.ReferenceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, utils.ReferenceItem{ID: row.ID, Name: row.Name})
	}

	return items
}
