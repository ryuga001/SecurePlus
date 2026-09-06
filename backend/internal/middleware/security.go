package middleware

import (
	"crypto/hmac"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
)

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func originOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}

func RequireOrigin(app config.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeMethod(c.Request.Method) {
			c.Next()
			return
		}

		candidate := c.GetHeader("Origin")
		if candidate == "" {
			candidate = originOf(c.GetHeader("Referer"))
		}

		if candidate == "" || candidate != app.AllowedOrigin {
			deny(c, http.StatusForbidden, "forbidden_origin", "origin not allowed")
			return
		}

		c.Next()
	}
}

func RequireCSRFCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeMethod(c.Request.Method) {
			c.Next()
			return
		}

		header := c.GetHeader("X-CSRF-Token")
		cookie, err := c.Cookie(auth.CookieCSRF)

		if header == "" || err != nil || cookie == "" || !hmac.Equal([]byte(header), []byte(cookie)) {
			deny(c, http.StatusForbidden, "csrf_failed", "csrf validation failed")
			return
		}

		c.Next()
	}
}

func RequireCSRF(cfg config.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeMethod(c.Request.Method) {
			c.Next()
			return
		}

		claims, ok := auth.ClaimsFrom(c)
		if !ok {
			deny(c, http.StatusForbidden, "csrf_failed", "csrf validation failed")
			return
		}

		header := c.GetHeader("X-CSRF-Token")
		expected := auth.CSRFToken(cfg.CSRFSecret, claims.ID)

		if header == "" || !hmac.Equal([]byte(header), []byte(expected)) {
			deny(c, http.StatusForbidden, "csrf_failed", "csrf validation failed")
			return
		}

		c.Next()
	}
}
