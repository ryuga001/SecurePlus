package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/profile"
	service "dpdp-backend/internal/admin/services/profile"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/middleware"
)

type ProfileHandler struct {
	svc *service.ProfileService
}

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

func (h *ProfileHandler) RegisterRoutes(protected *gin.RouterGroup, _ func(privilege string) gin.HandlerFunc) {
	profile := protected.Group("/admin/profile")

	profile.GET("", h.get)
	profile.PATCH("", h.update)
}

func (h *ProfileHandler) get(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	profile, err := h.svc.Get(c.Request.Context(), serviceActor(actor))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToProfileResponse(profile))
}

func (h *ProfileHandler) update(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	if req.AdminEmail != nil {
		utils.Respond(c, utils.ErrAdminEmailImmutable)
		return
	}

	input := service.ProfileInput{
		OrgName:   req.OrgName,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	profile, err := h.svc.Update(c.Request.Context(), serviceActor(actor), input)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToProfileResponse(profile))
}

func serviceActor(actor middleware.Actor) service.Actor {
	return service.Actor{
		CustomerID: actor.CustomerID,
		UserID:     actor.UserID,
		RoleID:     actor.RoleID,
	}
}
