package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"dpdp-backend/internal/db"
)

const msgAdminRoleRequired = "administrator role required"

func RequireAdminRole(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok || actor.RoleID == nil {
			deny(c, http.StatusForbidden, "forbidden", msgAdminRoleRequired)
			return
		}

		var types []string

		err := database.WithContext(c.Request.Context()).
			Model(&db.Role{}).
			Where("id = ?", *actor.RoleID).
			Pluck("type", &types).Error
		if err != nil {
			deny(c, http.StatusServiceUnavailable, "service_unavailable", "role lookup failed")
			return
		}

		if len(types) == 0 || types[0] != db.RoleTypeAdmin {
			deny(c, http.StatusForbidden, "forbidden", msgAdminRoleRequired)
			return
		}

		c.Next()
	}
}
