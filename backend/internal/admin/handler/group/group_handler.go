package group

import (
	"net/http"

	"github.com/gin-gonic/gin"

	emailuserdto "dpdp-backend/internal/admin/dto/emailuser"
	dto "dpdp-backend/internal/admin/dto/group"
	repo "dpdp-backend/internal/admin/repositories/group"
	service "dpdp-backend/internal/admin/services/group"
	"dpdp-backend/internal/admin/utils"
)

const (
	PrivilegeGroupView   = "admin.email.group.view"
	PrivilegeGroupCreate = "admin.email.group.create"
	PrivilegeGroupEdit   = "admin.email.group.edit"
	PrivilegeGroupDelete = "admin.email.group.delete"
)

type GroupHandler struct {
	svc *service.GroupService
}

func NewGroupHandler(svc *service.GroupService) *GroupHandler {
	return &GroupHandler{svc: svc}
}

func (h *GroupHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/groups")

	group.GET("", guard(PrivilegeGroupView), h.list)
	group.GET("/:id", guard(PrivilegeGroupView), h.get)
	group.POST("", guard(PrivilegeGroupCreate), h.create)
	group.PUT("/:id", guard(PrivilegeGroupEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeGroupDelete), h.remove)
	group.GET("/:id/users", guard(PrivilegeGroupView), h.members)
	group.POST("/:id/users", guard(PrivilegeGroupEdit), h.addMember)
	group.DELETE("/:id/users/:userID", guard(PrivilegeGroupEdit), h.removeMember)
}

func (h *GroupHandler) list(c *gin.Context) {
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

	items := make([]dto.GroupResponse, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, dto.FromGroup(item.Group, item.MemberCount))
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.GroupResponse]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *GroupHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	summary, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromGroup(summary.Group, summary.MemberCount))
}

func (h *GroupHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.GroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	summary, err := h.svc.Create(c.Request.Context(), actor.CustomerID, service.GroupInput{Name: req.Name, Type: req.Type})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromGroup(summary.Group, summary.MemberCount))
}

func (h *GroupHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.GroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	summary, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, service.GroupInput{Name: req.Name, Type: req.Type})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromGroup(summary.Group, summary.MemberCount))
}

func (h *GroupHandler) remove(c *gin.Context) {
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

func (h *GroupHandler) members(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var query utils.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.Members(c.Request.Context(), actor.CustomerID, id, repo.MemberListParams{
		Search:   query.Search,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, utils.ListResponse[emailuserdto.EmailUserResponse]{
		Items:    emailuserdto.FromEmailUsers(listing.Items),
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *GroupHandler) addMember(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.AssignUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	if err := h.svc.AddMember(c.Request.Context(), actor.CustomerID, id, req.EmailUserID); err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.MembershipResponse{GroupID: id, EmailUserID: req.EmailUserID})
}

func (h *GroupHandler) removeMember(c *gin.Context) {
	actor, id, userID, ok := utils.ActorAndIDs(c, "userID")
	if !ok {
		return
	}

	if err := h.svc.RemoveMember(c.Request.Context(), actor.CustomerID, id, userID); err != nil {
		utils.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
