package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email taken")
	ErrOrgNameTaken       = errors.New("org name taken")
	ErrNoPendingSignup    = errors.New("no pending signup")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidCode        = errors.New("invalid code")
	ErrCodeExhausted      = errors.New("code exhausted")
	ErrCooldown           = errors.New("cooldown active")
	ErrRateLimited        = errors.New("rate limited")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenReplayed      = errors.New("token replayed")
	ErrNoRole             = errors.New("no role assigned")
	ErrUnavailable        = errors.New("dependency unavailable")
	ErrEmailSendFailed    = errors.New("email send failed")
	ErrTemplateMissing    = errors.New("email template missing")
	ErrIdentityNotCached  = errors.New("identity not cached")
)
