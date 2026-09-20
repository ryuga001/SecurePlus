package emailuser

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/emailuser"
	groupdto "dpdp-backend/internal/admin/dto/group"
	repo "dpdp-backend/internal/admin/repositories/emailuser"
	service "dpdp-backend/internal/admin/services/emailuser"
	"dpdp-backend/internal/admin/utils"
)

const (
	PrivilegeEmailUserView   = "admin.email.user.view"
	PrivilegeEmailUserCreate = "admin.email.user.create"
	PrivilegeEmailUserEdit   = "admin.email.user.edit"
	PrivilegeEmailUserDelete = "admin.email.user.delete"
)

type EmailUserHandler struct {
	svc *service.EmailUserService
}

func NewEmailUserHandler(svc *service.EmailUserService) *EmailUserHandler {
	return &EmailUserHandler{svc: svc}
}

func (h *EmailUserHandler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/users")

	group.GET("", guard(PrivilegeEmailUserView), h.list)
	group.GET("/:id", guard(PrivilegeEmailUserView), h.get)
	group.POST("", guard(PrivilegeEmailUserCreate), h.create)
	group.PUT("/:id", guard(PrivilegeEmailUserEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeEmailUserDelete), h.remove)
	group.GET("/:id/groups", guard(PrivilegeEmailUserView), h.groups)
}

func (h *EmailUserHandler) list(c *gin.Context) {
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

	c.JSON(http.StatusOK, toEmailUserListResponse(listing))
}

func (h *EmailUserHandler) get(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromEmailUser(row))
}

func (h *EmailUserHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.EmailUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toEmailUserInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromEmailUser(row))
}

func (h *EmailUserHandler) update(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	var req dto.EmailUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	row, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toEmailUserInput(req))
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromEmailUser(row))
}

func (h *EmailUserHandler) remove(c *gin.Context) {
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

func (h *EmailUserHandler) groups(c *gin.Context) {
	actor, id, ok := utils.ActorAndID(c)
	if !ok {
		return
	}

	rows, err := h.svc.Groups(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]groupdto.GroupResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, groupdto.FromGroup(row, 0, make([]groupdto.MemberResponse, 0)))
	}

	c.JSON(http.StatusOK, utils.ListResponse[groupdto.GroupResponse]{
		Items:    items,
		Page:     utils.DefaultPage,
		PageSize: len(items),
		Total:    int64(len(items)),
	})
}

func toEmailUserInput(req dto.EmailUserRequest) service.EmailUserInput {
	return service.EmailUserInput{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}
}

func toEmailUserListResponse(listing service.Listing) utils.ListResponse[dto.EmailUserResponse] {
	return utils.ListResponse[dto.EmailUserResponse]{
		Items:    dto.FromEmailUsers(listing.Items),
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	}
}
