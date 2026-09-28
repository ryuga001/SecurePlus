package dashboarduser

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/dashboarduser"
	repo "dpdp-backend/internal/admin/repositories/dashboarduser"
	service "dpdp-backend/internal/admin/services/dashboarduser"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/middleware"
)

type DashboardUserHandler struct {
	svc *service.DashboardUserService
}

func NewDashboardUserHandler(svc *service.DashboardUserService) *DashboardUserHandler {
	return &DashboardUserHandler{svc: svc}
}

func (h *DashboardUserHandler) RegisterRoutes(protected *gin.RouterGroup, requireAdmin gin.HandlerFunc) {
	users := protected.Group("/admin/users", requireAdmin)

	users.GET("", h.list)
	users.GET("/:id", h.get)
	users.POST("", h.create)
	users.PUT("/:id", h.update)
	users.DELETE("/:id", h.remove)
}

func (h *DashboardUserHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query dto.UserListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.ListParams{
		Search:   query.Search,
		RoleID:   query.RoleID,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.UserResponse, 0, len(listing.Items))
	for _, row := range listing.Items {
		items = append(items, dto.ToUserResponse(row))
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.UserResponse]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *DashboardUserHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(row))
}

func (h *DashboardUserHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, invited, err := h.svc.Create(c.Request.Context(), serviceActor(actor), dto.ToInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateUserResponse{User: dto.ToUserResponse(row), InvitationSent: invited})
}

func (h *DashboardUserHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	if req.Email != nil {
		utils.Respond(c, utils.ErrUserEmailImmutable)
		return
	}

	row, err := h.svc.Update(c.Request.Context(), serviceActor(actor), id, dto.ToInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(row))
}

func (h *DashboardUserHandler) remove(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), serviceActor(actor), id); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func serviceActor(actor middleware.Actor) service.Actor {
	return service.Actor{CustomerID: actor.CustomerID, UserID: actor.UserID}
}
