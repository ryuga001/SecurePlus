package role

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/role"
	repo "dpdp-backend/internal/admin/repositories/role"
	service "dpdp-backend/internal/admin/services/role"
	"dpdp-backend/internal/admin/utils"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

func (h *RoleHandler) RegisterRoutes(protected *gin.RouterGroup, requireAdmin gin.HandlerFunc) {
	roles := protected.Group("/admin/roles", requireAdmin)

	roles.GET("", h.list)
	roles.GET("/options", h.options)
	roles.GET("/:id", h.get)
	roles.POST("", h.create)
	roles.PUT("/:id", h.update)
	roles.DELETE("/:id", h.remove)

	protected.GET("/admin/privileges", requireAdmin, h.privileges)
}

func (h *RoleHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query dto.RoleListQuery
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

	items := make([]dto.RoleListItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.ToRoleListItem(item))
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.RoleListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *RoleHandler) options(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	rows, err := h.svc.Options(c.Request.Context(), actor.CustomerID)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToRoleOptions(rows))
}

func (h *RoleHandler) privileges(c *gin.Context) {
	if _, ok := utils.Actor(c); !ok {
		return
	}

	names, err := h.svc.Privileges(c.Request.Context())
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.PrivilegesResponse{Privileges: names})
}

func (h *RoleHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	detail, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToRoleResponse(detail))
}

func (h *RoleHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToRoleResponse(detail))
}

func (h *RoleHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToRoleResponse(detail))
}

func (h *RoleHandler) remove(c *gin.Context) {
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

func toInput(req dto.RoleRequest) service.RoleInput {
	return service.RoleInput{
		Name:        req.Name,
		Description: req.Description,
		Privileges:  req.Privileges,
	}
}
