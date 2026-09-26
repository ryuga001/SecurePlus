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
	PrivilegeConfigurationView   = "admin.discovery.configuration.view"
	PrivilegeConfigurationCreate = "admin.discovery.configuration.create"
	PrivilegeConfigurationEdit   = "admin.discovery.configuration.edit"
	PrivilegeConfigurationDelete = "admin.discovery.configuration.delete"
	PrivilegeConfigurationTest   = "admin.discovery.configuration.test"
)

var configurationTypes = []string{
	db.ConfigurationTypeEntra,
	db.ConfigurationTypeAzureStorage,
	db.ConfigurationTypeGoogleSA,
	db.ConfigurationTypeAWSIAM,
}

var discoveryStatuses = []string{db.DiscoveryStatusActive, db.DiscoveryStatusInactive}

type ConfigurationHandler struct {
	svc *service.ConfigurationService
}

func NewConfigurationHandler(svc *service.ConfigurationService) *ConfigurationHandler {
	return &ConfigurationHandler{svc: svc}
}

func (h *ConfigurationHandler) RegisterRoutes(
	protected *gin.RouterGroup,
	guard func(privilege string) gin.HandlerFunc,
) {
	group := protected.Group("/admin/data-discovery/configurations")

	group.POST("/test", guard(PrivilegeConfigurationTest), h.test)
	group.GET("", guard(PrivilegeConfigurationView), h.list)
	group.GET("/:id", guard(PrivilegeConfigurationView), h.get)
	group.POST("", guard(PrivilegeConfigurationCreate), h.create)
	group.PUT("/:id", guard(PrivilegeConfigurationEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeConfigurationDelete), h.remove)
	group.GET("/:id/targets", guard(PrivilegePolicyView), h.targets)

	protected.GET("/admin/data-discovery/source-capabilities",
		guard(PrivilegeConfigurationView), h.capabilities)
}

func (h *ConfigurationHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query dto.ConfigurationListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.ConfigurationListParams{
		Search:             query.Search,
		ConfigurationTypes: utils.ParseEnumList(query.ConfigurationTypes, configurationTypes...),
		Statuses:           utils.ParseEnumList(query.Statuses, discoveryStatuses...),
		Page:               query.Page,
		PageSize:           query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.ConfigurationListItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.ConfigurationListItem{
			ID:                item.Configuration.ID,
			Name:              item.Configuration.Name,
			Description:       text(item.Configuration.Description),
			ConfigurationType: item.Configuration.ConfigurationType,
			Status:            item.Configuration.Status,
			PolicyCount:       item.PolicyCount,
			LastTestedAt:      item.Configuration.LastTestedAt,
			CreatedAt:         item.Configuration.CreatedAt,
			UpdatedAt:         item.Configuration.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.ConfigurationListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *ConfigurationHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	detail, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toConfigurationResponse(detail))
}

func (h *ConfigurationHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.ConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Create(c.Request.Context(), actor.CustomerID, service.ConfigurationInput{
		Name:              req.Name,
		Description:       req.Description,
		ConfigurationType: req.ConfigurationType,
		Status:            req.Status,
		Config:            req.Config,
		Secret:            req.Secret,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, toConfigurationResponse(detail))
}

func (h *ConfigurationHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.ConfigurationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, service.ConfigurationInput{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Config:      req.Config,
		Secret:      req.Secret,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toConfigurationResponse(detail))
}

func (h *ConfigurationHandler) remove(c *gin.Context) {
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

func (h *ConfigurationHandler) test(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.ConfigurationTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	err := h.svc.Test(c.Request.Context(), actor.CustomerID, service.TestInput{
		ConfigurationID: req.ConfigurationID,
		ConfigurationInput: service.ConfigurationInput{
			Name:              "connection test",
			ConfigurationType: req.ConfigurationType,
			Config:            req.Config,
			Secret:            req.Secret,
		},
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.TestResponse{Status: "ok"})
}

func (h *ConfigurationHandler) capabilities(c *gin.Context) {
	if _, ok := utils.Actor(c); !ok {
		return
	}

	rows, err := h.svc.Capabilities(c.Request.Context())
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.SourceCapabilityResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.SourceCapabilityResponse{
			ConfigurationType: row.ConfigurationType,
			SourceType:        row.SourceType,
			Label:             row.Label,
		})
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.SourceCapabilityResponse]{
		Items:    items,
		Page:     utils.DefaultPage,
		PageSize: len(items),
		Total:    int64(len(items)),
	})
}

func toConfigurationResponse(detail service.ConfigurationDetail) dto.ConfigurationResponse {
	return dto.ConfigurationResponse{
		ID:                detail.Configuration.ID,
		Name:              detail.Configuration.Name,
		Description:       text(detail.Configuration.Description),
		ConfigurationType: detail.Configuration.ConfigurationType,
		Config:            maskConfig(detail.Configuration.Config),
		Status:            detail.Configuration.Status,
		HasCredential:     detail.HasCredential,
		PolicyCount:       detail.PolicyCount,
		LastTestedAt:      detail.Configuration.LastTestedAt,
		CreatedAt:         detail.Configuration.CreatedAt,
		UpdatedAt:         detail.Configuration.UpdatedAt,
	}
}

func maskConfig(config db.StringMap) map[string]string {
	masked := make(map[string]string, len(config))

	for key, value := range config {
		if key == "accessKeyId" {
			masked[key] = maskIdentifier(value)
			continue
		}

		masked[key] = value
	}

	return masked
}

func maskIdentifier(value string) string {
	if len(value) <= 8 {
		return "****"
	}

	return value[:4] + "****" + value[len(value)-4:]
}

func text(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func (h *ConfigurationHandler) targets(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var query dto.TargetBrowseQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.Targets(c.Request.Context(), actor.CustomerID, id, service.TargetInput{
		SourceType: query.SourceType,
		Search:     query.Search,
		Parent:     query.Parent,
		Cursor:     query.Cursor,
		Limit:      query.Limit,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.TargetOptionItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.TargetOptionItem{
			Value:       item.Value,
			Label:       item.Label,
			Description: item.Description,
			Kind:        item.Kind,
			Expandable:  item.Expandable,
		})
	}

	c.JSON(http.StatusOK, dto.TargetBrowseResponse{Items: items, NextCursor: listing.NextCursor})
}
