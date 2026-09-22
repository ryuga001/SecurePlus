package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"dpdp-backend/internal/auth"
)

func RequirePrivilege(database *gorm.DB, store *auth.Store, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok || actor.RoleID == nil {
			deny(c, http.StatusForbidden, "forbidden", "no privileges assigned")
			return
		}

		granted, err := auth.PrivilegeNames(c.Request.Context(), database, store, *actor.RoleID)
		if err != nil {
			deny(c, http.StatusServiceUnavailable, "service_unavailable", "privilege lookup failed")
			return
		}

		for _, item := range granted {
			if item == name {
				c.Next()
				return
			}
		}

		deny(c, http.StatusForbidden, "forbidden", "missing privilege "+name)
	}
}
