package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
)

type Actor struct {
	UserID     int
	CustomerID int
	RoleID     *int
}

func ActorFrom(c *gin.Context) (Actor, bool) {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return Actor{}, false
	}

	return Actor{
		UserID:     auth.UserID(claims),
		CustomerID: claims.CustomerID,
		RoleID:     claims.RoleID,
	}, true
}

func RequireAuth(svc *auth.Service, store *auth.Store, cfg config.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(auth.CookieAccess)
		if err != nil || raw == "" {
			deny(c, http.StatusUnauthorized, "unauthenticated", "missing access token")
			return
		}

		claims, err := auth.Verify(c.Request.Context(), raw, cfg.Issuer, auth.TypeAccess, svc.TenantSecret)
		if errors.Is(err, auth.ErrUnavailable) {
			deny(c, http.StatusServiceUnavailable, "service_unavailable", "authentication store unavailable")
			return
		}

		if err != nil {
			deny(c, http.StatusUnauthorized, "unauthenticated", "invalid access token")
			return
		}

		blocked, version, err := store.State(c.Request.Context(), claims.ID, auth.UserID(claims))
		if err != nil {
			deny(c, http.StatusServiceUnavailable, "service_unavailable", "authentication store unavailable")
			return
		}

		if blocked || claims.Version < version {
			deny(c, http.StatusUnauthorized, "unauthenticated", "token revoked")
			return
		}

		auth.SetClaims(c, claims)
		c.Next()
	}
}

func deny(c *gin.Context, status int, code, message string) {
	if status == http.StatusServiceUnavailable {
		c.Header("Retry-After", "5")
	}

	c.AbortWithStatusJSON(status, gin.H{"error": code, "message": message})
}
