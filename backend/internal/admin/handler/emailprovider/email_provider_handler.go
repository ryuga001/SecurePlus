package emailprovider

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/emailprovider"
	repo "dpdp-backend/internal/admin/repositories/emailprovider"
	service "dpdp-backend/internal/admin/services/emailprovider"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	PrivilegeProviderView   = "admin.email.provider.view"
	PrivilegeProviderCreate = "admin.email.provider.create"
	PrivilegeProviderEdit   = "admin.email.provider.edit"
	PrivilegeProviderDelete = "admin.email.provider.delete"
)

const DKIMSelector = "dpdp"

type EmailProviderHandler struct {
	svc *service.EmailProviderService
}

func NewEmailProviderHandler(svc *service.EmailProviderService) *EmailProviderHandler {
	return &EmailProviderHandler{svc: svc}
}

func (h *EmailProviderHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/configurations")

	group.GET("", guard(PrivilegeProviderView), h.list)
	group.GET("/:id", guard(PrivilegeProviderView), h.get)
	group.POST("", guard(PrivilegeProviderCreate), h.create)
	group.PUT("/:id", guard(PrivilegeProviderEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeProviderDelete), h.remove)
	group.POST("/:id/dkim", guard(PrivilegeProviderEdit), h.dkim)
	group.POST("/:id/access-token", guard(PrivilegeProviderEdit), h.accessToken)
}

func (h *EmailProviderHandler) list(c *gin.Context) {
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

	items := make([]dto.ConfigurationResponse, 0, len(listing.Items))
	for _, row := range listing.Items {
		items = append(items, toConfigurationResponse(row))
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.ConfigurationResponse]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *EmailProviderHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toConfigurationResponse(row))
}

func (h *EmailProviderHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.ConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toConfigurationInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, toConfigurationResponse(row))
}

func (h *EmailProviderHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.ConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toConfigurationInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toConfigurationResponse(row))
}

func (h *EmailProviderHandler) remove(c *gin.Context) {
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

func (h *EmailProviderHandler) dkim(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.GenerateDKIM(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	public := ""
	if row.DKIMPublicKey != nil {
		public = *row.DKIMPublicKey
	}

	private := ""
	if row.DKIMPrivateKey != nil {
		private = *row.DKIMPrivateKey
	}

	c.JSON(http.StatusOK, dto.DKIMResponse{
		Selector:       DKIMSelector,
		RecordName:     DKIMSelector + "._domainkey." + row.Domain,
		DKIMPublicKey:  public,
		DKIMPrivateKey: private,
	})
}

func (h *EmailProviderHandler) accessToken(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	token, expiresAt, err := h.svc.GenerateAccessToken(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AccessTokenResponse{AccessToken: token, ExpiresAt: expiresAt})
}

func toConfigurationInput(req dto.ConfigurationRequest) service.ConfigurationInput {
	return service.ConfigurationInput{Name: req.Name, Domain: req.Domain, Provider: req.Provider}
}

func toConfigurationResponse(row db.EmailProviderConfiguration) dto.ConfigurationResponse {
	return dto.ConfigurationResponse{
		ID:                   row.ID,
		Name:                 row.Name,
		Domain:               row.Domain,
		Provider:             row.Provider,
		DKIMPublicKey:        row.DKIMPublicKey,
		HasAccessToken:       row.AccessTokenExpiresAt != nil,
		AccessTokenExpiresAt: row.AccessTokenExpiresAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}
