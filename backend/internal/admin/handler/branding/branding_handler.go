package branding

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/branding"
	service "dpdp-backend/internal/admin/services/branding"
	"dpdp-backend/internal/admin/utils"
)

const PrivilegeBrandingEdit = "admin.branding.edit"

const multipartOverhead = 64 << 10

type BrandingHandler struct {
	svc *service.BrandingService
}

func NewBrandingHandler(svc *service.BrandingService) *BrandingHandler {
	return &BrandingHandler{svc: svc}
}

func (h *BrandingHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	branding := protected.Group("/admin/branding")

	branding.PATCH("", guard(PrivilegeBrandingEdit), h.update)
	branding.POST("/logo", guard(PrivilegeBrandingEdit), h.uploadLogo)
	branding.DELETE("/logo", guard(PrivilegeBrandingEdit), h.removeLogo)
}

func (h *BrandingHandler) update(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.UpdateBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	input := service.BrandingInput{
		Theme:    req.Theme,
		Language: req.Language,
		Timezone: req.Timezone,
	}

	if err := h.svc.Update(c.Request.Context(), actor.CustomerID, input); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *BrandingHandler) uploadLogo(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxLogoSize+multipartOverhead)

	header, err := c.FormFile("logo")
	if err != nil {
		utils.Respond(c, utils.ErrInvalidLogo)
		return
	}

	if err := h.svc.UploadLogo(c.Request.Context(), actor.CustomerID, header); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *BrandingHandler) removeLogo(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	if err := h.svc.RemoveLogo(c.Request.Context(), actor.CustomerID); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
