package utils

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/middleware"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type ListQuery struct {
	Search   string `form:"search" binding:"max=100"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type ListResponse[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type ReferenceItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: code, Message: message})
}

func BadRequest(c *gin.Context) {
	Fail(c, http.StatusBadRequest, CodeValidation, MsgInvalidRequest)
}

func Actor(c *gin.Context) (middleware.Actor, bool) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		Respond(c, ErrUnauthenticated)
		return middleware.Actor{}, false
	}

	return actor, true
}

func ActorAndID(c *gin.Context) (middleware.Actor, int, bool) {
	actor, ok := Actor(c)
	if !ok {
		return middleware.Actor{}, 0, false
	}

	id, ok := PathID(c, "id")
	if !ok {
		return middleware.Actor{}, 0, false
	}

	return actor, id, true
}

func ActorAndIDs(c *gin.Context, second string) (middleware.Actor, int, int, bool) {
	actor, id, ok := ActorAndID(c)
	if !ok {
		return middleware.Actor{}, 0, 0, false
	}

	other, ok := PathID(c, second)
	if !ok {
		return middleware.Actor{}, 0, 0, false
	}

	return actor, id, other, true
}

func PathID(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id < 1 {
		Fail(c, http.StatusBadRequest, CodeValidation, MsgInvalidIdentifier)
		return 0, false
	}

	return id, true
}

func Respond(c *gin.Context, err error) {
	status, code := statusFor(err)

	if status == http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "admin request failed", "error", err)
		Fail(c, status, code, MsgUnexpected)

		return
	}

	Fail(c, status, code, err.Error())
}

func statusFor(err error) (int, string) {
	switch {
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized, CodeUnauthenticated
	case errors.Is(err, ErrUnavailable):
		return http.StatusServiceUnavailable, CodeUnavailable

	case errors.Is(err, ErrConfigurationNotFound),
		errors.Is(err, ErrPolicyNotFound),
		errors.Is(err, ErrRuleNotFound),
		errors.Is(err, ErrGroupNotFound),
		errors.Is(err, ErrEmailUserNotFound),
		errors.Is(err, ErrMappingNotFound),
		errors.Is(err, ErrBrandingNotFound):
		return http.StatusNotFound, CodeNotFound

	case errors.Is(err, ErrDomainTaken):
		return http.StatusConflict, CodeDomainTaken
	case errors.Is(err, ErrPolicyNameTaken):
		return http.StatusConflict, CodePolicyNameTaken
	case errors.Is(err, ErrRuleNameTaken):
		return http.StatusConflict, CodeRuleNameTaken
	case errors.Is(err, ErrGroupNameTaken):
		return http.StatusConflict, CodeGroupNameTaken
	case errors.Is(err, ErrEmailTaken):
		return http.StatusConflict, CodeEmailTaken
	case errors.Is(err, ErrMappingExists):
		return http.StatusConflict, CodeMappingExists

	case errors.Is(err, ErrInvalidDomain):
		return http.StatusBadRequest, CodeInvalidDomain
	case errors.Is(err, ErrInvalidProvider):
		return http.StatusBadRequest, CodeInvalidProvider
	case errors.Is(err, ErrInvalidRegex):
		return http.StatusBadRequest, CodeInvalidRegex
	case errors.Is(err, ErrInvalidRuleType):
		return http.StatusBadRequest, CodeInvalidRuleType
	case errors.Is(err, ErrInvalidGroupType):
		return http.StatusBadRequest, CodeInvalidGroupType
	case errors.Is(err, ErrInvalidEmail):
		return http.StatusBadRequest, CodeInvalidEmail

	case errors.Is(err, ErrUnknownGroup):
		return http.StatusBadRequest, CodeGroupNotFound
	case errors.Is(err, ErrUnknownRule):
		return http.StatusBadRequest, CodeRuleNotFound
	case errors.Is(err, ErrUnknownEmailUser):
		return http.StatusBadRequest, CodeEmailUserNotFound

	case errors.Is(err, ErrInvalidTheme):
		return http.StatusBadRequest, CodeInvalidTheme
	case errors.Is(err, ErrInvalidLanguage):
		return http.StatusBadRequest, CodeInvalidLanguage
	case errors.Is(err, ErrInvalidTimezone):
		return http.StatusBadRequest, CodeInvalidTimezone
	case errors.Is(err, ErrInvalidLogo):
		return http.StatusBadRequest, CodeInvalidLogo

	case errors.Is(err, ErrInvalidAction),
		errors.Is(err, ErrInvalidRestrictionMode),
		errors.Is(err, ErrRestrictionValuesNeeded),
		errors.Is(err, ErrInvalidRestrictionDomain),
		errors.Is(err, ErrInvalidRestrictionFileType),
		errors.Is(err, ErrPolicyNameNeeded),
		errors.Is(err, ErrRulesNeeded),
		errors.Is(err, ErrGroupsNeeded),
		errors.Is(err, ErrRuleNameNeeded),
		errors.Is(err, ErrRuleValueNeeded),
		errors.Is(err, ErrGroupNameNeeded),
		errors.Is(err, ErrNameNeeded),
		errors.Is(err, ErrTooManyItems):
		return http.StatusBadRequest, CodeValidation
	}

	return http.StatusInternalServerError, CodeInternal
}
