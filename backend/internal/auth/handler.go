package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/config"
)

const claimsContextKey = "dpdp.claims"

func SetClaims(c *gin.Context, claims *Claims) {
	c.Set(claimsContextKey, claims)
}

func ClaimsFrom(c *gin.Context) (*Claims, bool) {
	value, exists := c.Get(claimsContextKey)
	if !exists {
		return nil, false
	}

	claims, ok := value.(*Claims)

	return claims, ok
}

const (
	CookieAccess  = "dpdp_at"
	CookieRefresh = "dpdp_rt"
	CookieCSRF    = "dpdp_csrf"

	RefreshPath = "/api/v1/auth"
	csrfTTL     = 30 * 24 * time.Hour
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,max=128"`
}

type CompleteRequest struct {
	RegistrationToken string `json:"registration_token" binding:"required"`
	OrgName           string `json:"org_name" binding:"required,min=2,max=100"`
	FirstName         string `json:"first_name" binding:"required,min=1,max=50"`
	LastName          string `json:"last_name" binding:"required,min=1,max=50"`
	Password          string `json:"password" binding:"required,min=10,max=128"`
}

type VerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6,number"`
}

type verifiedResponse struct {
	RegistrationToken string `json:"registration_token"`
	ExpiresIn         int    `json:"expires_in"`
}

type EmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetRequest struct {
	Email       string `json:"email" binding:"required,email"`
	ResetCode   string `json:"reset_code" binding:"required,len=6,number"`
	NewPassword string `json:"new_password" binding:"required,min=10,max=128"`
}

type messageResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in,omitempty"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type privilegesResponse struct {
	Privileges []string `json:"privileges"`
}

type Handler struct {
	svc *Service
	cfg config.Auth
}

func NewHandler(svc *Service, cfg config.Auth) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) RegisterRoutes(public, refresh, protected *gin.RouterGroup) {
	public.GET("/auth/csrf", h.csrf)
	public.POST("/auth/login", h.login)
	public.POST("/auth/register/start", h.startRegistration)
	public.POST("/auth/register/verify", h.verifyEmail)
	public.POST("/auth/register/resend", h.resendCode)
	public.POST("/auth/register/complete", h.completeRegistration)
	public.POST("/auth/password/forgot", h.forgotPassword)
	public.POST("/auth/password/reset", h.resetPassword)

	refresh.POST("/auth/refresh", h.refresh)

	protected.GET("/me", h.me)
	protected.GET("/me/privileges", h.privileges)
	protected.POST("/auth/logout", h.logout)
}

func CSRFToken(secret, id string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *Handler) setCookie(c *gin.Context, name, value, path string, ttl time.Duration, httpOnly bool, sameSite http.SameSite) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(ttl.Seconds()),
		Secure:   h.cfg.CookieSecure,
		HttpOnly: httpOnly,
		SameSite: sameSite,
	}

	if ttl <= 0 {
		cookie.Expires = time.Unix(0, 0)
	}

	http.SetCookie(c.Writer, cookie)
}

func (h *Handler) refreshSameSite() http.SameSite {
	if h.cfg.CookieSameSite == http.SameSiteNoneMode {
		return http.SameSiteNoneMode
	}

	return http.SameSiteStrictMode
}

func (h *Handler) applyPair(c *gin.Context, pair TokenPair) string {
	token := CSRFToken(h.cfg.CSRFSecret, pair.Access.ID)

	h.setCookie(c, CookieAccess, pair.Access.Token, "/", time.Until(pair.Access.ExpiresAt), true, h.cfg.CookieSameSite)
	h.setCookie(c, CookieRefresh, pair.Refresh.Token, RefreshPath, time.Until(pair.Refresh.ExpiresAt), true, h.refreshSameSite())
	h.setCookie(c, CookieCSRF, token, "/", csrfTTL, false, h.cfg.CookieSameSite)

	return token
}

func (h *Handler) clearCookies(c *gin.Context) {
	h.setCookie(c, CookieAccess, "", "/", -time.Hour, true, h.cfg.CookieSameSite)
	h.setCookie(c, CookieRefresh, "", RefreshPath, -time.Hour, true, h.refreshSameSite())
	h.setCookie(c, CookieCSRF, "", "/", -time.Hour, false, h.cfg.CookieSameSite)
}

func (h *Handler) csrf(c *gin.Context) {
	id, err := NewSalt()
	if err != nil {
		respond(c, err)
		return
	}

	token := CSRFToken(h.cfg.CSRFSecret, id)
	h.setCookie(c, CookieCSRF, token, "/", csrfTTL, false, h.cfg.CookieSameSite)

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	snapshot, pair, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, Identity{
		IdentitySnapshot: snapshot,
		CSRFToken:        h.applyPair(c, pair),
	})
}

func (h *Handler) startRegistration(c *gin.Context) {
	var req EmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	ttl, err := h.svc.StartRegistration(c.Request.Context(), req.Email)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusAccepted, messageResponse{
		Message:   "Verification code sent",
		ExpiresIn: int(ttl.Seconds()),
	})
}

func (h *Handler) verifyEmail(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	token, ttl, err := h.svc.VerifyEmail(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, verifiedResponse{
		RegistrationToken: token,
		ExpiresIn:         int(ttl.Seconds()),
	})
}

func (h *Handler) completeRegistration(c *gin.Context) {
	var req CompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	snapshot, pair, err := h.svc.CompleteRegistration(c.Request.Context(), req)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, Identity{
		IdentitySnapshot: snapshot,
		CSRFToken:        h.applyPair(c, pair),
	})
}

func (h *Handler) resendCode(c *gin.Context) {
	var req EmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	ttl, err := h.svc.ResendCode(c.Request.Context(), req.Email)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusAccepted, messageResponse{
		Message:   "Verification code sent",
		ExpiresIn: int(ttl.Seconds()),
	})
}

func (h *Handler) forgotPassword(c *gin.Context) {
	var req EmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	ttl, err := h.svc.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusAccepted, messageResponse{
		Message:   "If the address is registered, a reset code has been sent",
		ExpiresIn: int(ttl.Seconds()),
	})
}

func (h *Handler) resetPassword(c *gin.Context) {
	var req ResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c)
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), req.Email, req.ResetCode, req.NewPassword); err != nil {
		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, messageResponse{Message: "Password updated"})
}

func (h *Handler) refresh(c *gin.Context) {
	raw, err := c.Cookie(CookieRefresh)
	if err != nil || raw == "" {
		respond(c, ErrInvalidToken)
		return
	}

	snapshot, pair, err := h.svc.Refresh(c.Request.Context(), raw)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenReplayed) {
			h.clearCookies(c)
		}

		respond(c, err)
		return
	}

	c.JSON(http.StatusOK, Identity{
		IdentitySnapshot: snapshot,
		CSRFToken:        h.applyPair(c, pair),
	})
}

func (h *Handler) logout(c *gin.Context) {
	claims, ok := ClaimsFrom(c)
	if !ok {
		respond(c, ErrInvalidToken)
		return
	}

	refreshID, refreshExp := "", time.Time{}
	if raw, err := c.Cookie(CookieRefresh); err == nil && raw != "" {
		if rc, err := Verify(c.Request.Context(), raw, h.cfg.Issuer, TypeRefresh, h.svc.TenantSecret); err == nil {
			refreshID, refreshExp = rc.ID, rc.ExpiresAt.Time
		}
	}

	if err := h.svc.Logout(c.Request.Context(), claims.ID, refreshID, claims.ExpiresAt.Time, refreshExp); err != nil {
		respond(c, err)
		return
	}

	h.clearCookies(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) me(c *gin.Context) {
	claims, ok := ClaimsFrom(c)
	if !ok {
		respond(c, ErrInvalidToken)
		return
	}

	snapshot, err := h.svc.Identity(c.Request.Context(), UserID(claims))
	if err != nil {
		respond(c, ErrInvalidToken)
		return
	}

	c.JSON(http.StatusOK, Identity{
		IdentitySnapshot: snapshot,
		CSRFToken:        CSRFToken(h.cfg.CSRFSecret, claims.ID),
	})
}

func (h *Handler) privileges(c *gin.Context) {
	claims, ok := ClaimsFrom(c)
	if !ok {
		respond(c, ErrInvalidToken)
		return
	}

	if claims.RoleID == nil {
		c.JSON(http.StatusOK, privilegesResponse{Privileges: []string{}})
		return
	}

	names, err := h.svc.Privileges(c.Request.Context(), *claims.RoleID)
	if err != nil {
		respond(c, ErrUnavailable)
		return
	}

	c.JSON(http.StatusOK, privilegesResponse{Privileges: names})
}

func badRequest(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{
		Error:   "validation_failed",
		Message: "request body is invalid",
	})
}

func respond(c *gin.Context, err error) {
	status, code := http.StatusInternalServerError, "internal_error"

	switch {
	case errors.Is(err, ErrInvalidCredentials):
		status, code = http.StatusUnauthorized, "invalid_credentials"
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenReplayed):
		status, code = http.StatusUnauthorized, "unauthenticated"
	case errors.Is(err, ErrInvalidCode):
		status, code = http.StatusUnauthorized, "invalid_code"
	case errors.Is(err, ErrNoRole):
		status, code = http.StatusForbidden, "no_role"
	case errors.Is(err, ErrEmailTaken):
		status, code = http.StatusConflict, "email_taken"
	case errors.Is(err, ErrOrgNameTaken):
		status, code = http.StatusConflict, "org_name_taken"
	case errors.Is(err, ErrNoPendingSignup):
		status, code = http.StatusNotFound, "no_pending_signup"
	case errors.Is(err, ErrEmailNotVerified):
		status, code = http.StatusUnauthorized, "email_not_verified"
	case errors.Is(err, ErrCooldown):
		status, code = http.StatusTooManyRequests, "cooldown"
	case errors.Is(err, ErrRateLimited):
		status, code = http.StatusTooManyRequests, "rate_limited"
	case errors.Is(err, ErrCodeExhausted):
		status, code = http.StatusTooManyRequests, "code_exhausted"
	case errors.Is(err, ErrEmailSendFailed):
		status, code = http.StatusBadGateway, "email_send_failed"
	case errors.Is(err, ErrTemplateMissing):
		status, code = http.StatusInternalServerError, "email_template_missing"
	case errors.Is(err, ErrUnavailable):
		status, code = http.StatusServiceUnavailable, "service_unavailable"
	}

	c.AbortWithStatusJSON(status, errorResponse{Error: code, Message: err.Error()})
}
