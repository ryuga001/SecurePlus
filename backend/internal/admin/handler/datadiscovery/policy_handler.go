package datadiscovery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/datadiscovery"
	repo "dpdp-backend/internal/admin/repositories/datadiscovery"
	service "dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	PrivilegePolicyView   = "admin.discovery.policy.view"
	PrivilegePolicyCreate = "admin.discovery.policy.create"
	PrivilegePolicyEdit   = "admin.discovery.policy.edit"
	PrivilegePolicyDelete = "admin.discovery.policy.delete"
)

var sourceTypes = []string{
	db.SourceTypeSharePoint,
	db.SourceTypeOneDrive,
	db.SourceTypeAzureBlob,
	db.SourceTypeGoogleDrive,
	db.SourceTypeAWSS3,
}

type PolicyHandler struct {
	svc *service.PolicyService
}

func NewPolicyHandler(svc *service.PolicyService) *PolicyHandler {
	return &PolicyHandler{svc: svc}
}

func (h *PolicyHandler) RegisterRoutes(
	protected *gin.RouterGroup,
	guard func(privilege string) gin.HandlerFunc,
) {
	group := protected.Group("/admin/data-discovery/policies")

	group.GET("", guard(PrivilegePolicyView), h.list)
	group.GET("/:id", guard(PrivilegePolicyView), h.get)
	group.POST("", guard(PrivilegePolicyCreate), h.create)
	group.PUT("/:id", guard(PrivilegePolicyEdit), h.update)
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

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.PolicyListParams{
		Search:           query.Search,
		SourceTypes:      utils.ParseEnumList(query.SourceTypes, sourceTypes...),
		ConfigurationIDs: utils.ParseIDList(query.ConfigurationIDs),
		Statuses:         utils.ParseEnumList(query.Statuses, discoveryStatuses...),
		Page:             query.Page,
		PageSize:         query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.PolicyListItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.PolicyListItem{
			ID:                item.Policy.ID,
			Name:              item.Policy.Name,
			Description:       text(item.Policy.Description),
			ConfigurationID:   item.Policy.ConfigurationID,
			ConfigurationName: item.Configuration.Name,
			ConfigurationType: item.Policy.ConfigurationType,
			SourceType:        item.Policy.SourceType,
			Status:            item.Policy.Status,
			TargetCount:       len(item.Policy.TargetList),
			FileTypeCount:     len(item.Policy.FileTypes),
			RuleCount:         item.RuleCount,
			CreatedAt:         item.Policy.CreatedAt,
			UpdatedAt:         item.Policy.UpdatedAt,
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
	return service.PolicyInput{
		Name:            req.Name,
		Description:     req.Description,
		SourceType:      req.SourceType,
		Status:          req.Status,
		ConfigurationID: req.ConfigurationID,
		TargetList:      req.TargetList,
		FileTypes:       req.FileTypes,
		RuleIDs:         req.RuleIDs,
	}
}

func toPolicyResponse(detail service.PolicyDetail) dto.PolicyResponse {
	targets := []string(detail.Policy.TargetList)
	if targets == nil {
		targets = []string{}
	}

	fileTypes := []string(detail.Policy.FileTypes)
	if fileTypes == nil {
		fileTypes = []string{}
	}

	rules := make([]utils.ReferenceItem, 0, len(detail.Rules))
	for _, rule := range detail.Rules {
		rules = append(rules, utils.ReferenceItem{ID: rule.ID, Name: rule.Name})
	}

	return dto.PolicyResponse{
		ID:                detail.Policy.ID,
		Name:              detail.Policy.Name,
		Description:       text(detail.Policy.Description),
		ConfigurationID:   detail.Policy.ConfigurationID,
		ConfigurationName: detail.Configuration.Name,
		ConfigurationType: detail.Policy.ConfigurationType,
		SourceType:        detail.Policy.SourceType,
		Status:            detail.Policy.Status,
		TargetList:        targets,
		FileTypes:         fileTypes,
		Rules:             rules,
		CreatedAt:         detail.Policy.CreatedAt,
		UpdatedAt:         detail.Policy.UpdatedAt,
	}
}
