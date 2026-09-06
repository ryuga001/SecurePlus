package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

func RequirePrivilege(database *gorm.DB, store *auth.Store, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok || actor.RoleID == nil {
			deny(c, http.StatusForbidden, "forbidden", "no privileges assigned")
			return
		}

		granted, err := privilegeNames(c, database, store, *actor.RoleID)
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

func privilegeNames(c *gin.Context, database *gorm.DB, store *auth.Store, roleID int) ([]string, error) {
	ctx := c.Request.Context()

	cached, err := store.Privileges(ctx, roleID)
	if err == nil && len(cached) > 0 {
		return cached, nil
	}
	if err != nil {
		slog.WarnContext(ctx, "privilege cache unavailable", "error", err)
	}

	var names []string

	query := database.WithContext(ctx).
		Model(&db.Privilege{}).
		Select("privileges.name").
		Joins("JOIN role_privileges rp ON rp.privilege_id = privileges.id").
		Where("rp.role_id = ?", roleID)

	if err := query.Scan(&names).Error; err != nil {
		return nil, err
	}

	if cacheErr := store.CachePrivileges(ctx, roleID, names, config.PrivCacheTTL); cacheErr != nil {
		slog.WarnContext(ctx, "privilege cache write failed", "error", cacheErr)
	}

	return names, nil
}
