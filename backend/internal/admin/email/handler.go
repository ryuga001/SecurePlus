package email

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/middleware"
)

const (
	PrivilegeView   = "admin.email.provider.view"
	PrivilegeCreate = "admin.email.provider.create"
	PrivilegeEdit   = "admin.email.provider.edit"
	PrivilegeDelete = "admin.email.provider.delete"
)

const DKIMSelector = "dpdp"

type ConfigurationRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Domain   string `json:"domain" binding:"required,max=253"`
	Provider string `json:"provider" binding:"required,max=20"`
}

type ListQuery struct {
	Search   string `form:"search" binding:"max=100"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type ConfigurationResponse struct {
	ID                   int        `json:"id"`
	Name                 string     `json:"name"`
	Domain               string     `json:"domain"`
	Provider             string     `json:"provider"`
	DKIMPublicKey        *string    `json:"dkim_public_key"`
	HasAccessToken       bool       `json:"has_access_token"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type ListResponse struct {
	Items    []ConfigurationResponse `json:"items"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
	Total    int64                   `json:"total"`
}

type DKIMResponse struct {
	Selector      string `json:"selector"`
	RecordName    string `json:"record_name"`
	DKIMPublicKey string `json:"dkim_public_key"`
}

type AccessTokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup, guard func(privilege string) gin.HandlerFunc) {
	group := protected.Group("/admin/email/configurations")

	group.GET("", guard(PrivilegeView), h.list)
	group.GET("/:id", guard(PrivilegeView), h.get)
	group.POST("", guard(PrivilegeCreate), h.create)
	group.PUT("/:id", guard(PrivilegeEdit), h.update)
	group.DELETE("/:id", guard(PrivilegeDelete), h.remove)
	group.POST("/:id/dkim", guard(PrivilegeEdit), h.dkim)
	group.POST("/:id/access-token", guard(PrivilegeEdit), h.accessToken)
}

func (h *Handler) list(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		respond(c, ErrUnauthenticated)
		return
	}

	var query ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		badRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, ListParams{
		Search:   query.Search,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toListResponse(listing))
}

func (h *Handler) get(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(row))
}

func (h *Handler) create(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		respond(c, ErrUnauthenticated)
		return
	}

	var req ConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	row, err := h.svc.Create(c.Request.Context(), actor.CustomerID, toInput(req))
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(row))
}

func (h *Handler) update(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}

	var req ConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	row, err := h.svc.Update(c.Request.Context(), actor.CustomerID, id, toInput(req))
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(row))
}

func (h *Handler) remove(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), actor.CustomerID, id); err != nil {
		respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) dkim(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}

	row, err := h.svc.GenerateDKIM(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		respond(c, err)
		return
	}

	key := ""
	if row.DKIMPublicKey != nil {
		key = *row.DKIMPublicKey
	}

	c.JSON(http.StatusOK, DKIMResponse{
		Selector:      DKIMSelector,
		RecordName:    DKIMSelector + "._domainkey." + row.Domain,
		DKIMPublicKey: key,
	})
}

func (h *Handler) accessToken(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}

	token, expiresAt, err := h.svc.GenerateAccessToken(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, AccessTokenResponse{AccessToken: token, ExpiresAt: expiresAt})
}

func actorAndID(c *gin.Context) (middleware.Actor, int, bool) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		respond(c, ErrUnauthenticated)
		return middleware.Actor{}, 0, false
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{
			Error:   utils.CodeValidation,
			Message: utils.MsgInvalidIdentifier,
		})

		return middleware.Actor{}, 0, false
	}

	return actor, id, true
}

func toInput(req ConfigurationRequest) Input {
	return Input{Name: req.Name, Domain: req.Domain, Provider: req.Provider}
}

func toResponse(row db.EmailProviderConfiguration) ConfigurationResponse {
	return ConfigurationResponse{
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

func toListResponse(listing Listing) ListResponse {
	items := make([]ConfigurationResponse, 0, len(listing.Items))
	for _, row := range listing.Items {
		items = append(items, toResponse(row))
	}

	return ListResponse{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	}
}

func badRequest(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{
		Error:   utils.CodeValidation,
		Message: utils.MsgInvalidRequest,
	})
}

func respond(c *gin.Context, err error) {
	status, code := http.StatusInternalServerError, utils.CodeInternal

	switch {
	case errors.Is(err, ErrNotFound):
		status, code = http.StatusNotFound, utils.CodeNotFound
	case errors.Is(err, ErrDomainTaken):
		status, code = http.StatusConflict, utils.CodeDomainTaken
	case errors.Is(err, ErrInvalidDomain):
		status, code = http.StatusBadRequest, utils.CodeInvalidDomain
	case errors.Is(err, ErrInvalidProvider):
		status, code = http.StatusBadRequest, utils.CodeInvalidProvider
	case errors.Is(err, ErrUnauthenticated):
		status, code = http.StatusUnauthorized, utils.CodeUnauthenticated
	case errors.Is(err, ErrUnavailable):
		status, code = http.StatusServiceUnavailable, utils.CodeUnavailable
	}

	if status == http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "email provider request failed", "error", err)
		c.AbortWithStatusJSON(status, errorResponse{Error: code, Message: utils.MsgUnexpected})

		return
	}

	c.AbortWithStatusJSON(status, errorResponse{Error: code, Message: err.Error()})
}
