package branding

import (
	"bytes"
	"context"
	"image"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	repo "dpdp-backend/internal/admin/repositories/branding"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/db"
	"dpdp-backend/internal/storage"
)

const (
	MaxLogoSize = 1 << 20

	maxLogoDimension = 4096
)

type IdentityCache interface {
	DropCustomerIdentities(ctx context.Context, customerID int) error
}

type BrandingInput struct {
	Theme    *string
	Language *string
	Timezone *string
}

type BrandingService struct {
	repo           *repo.BrandingRepository
	storage        *storage.Storage
	identityCache  IdentityCache
	logoPresignTTL time.Duration
}

func NewBrandingService(
	repository *repo.BrandingRepository,
	store *storage.Storage,
	identityCache IdentityCache,
	logoPresignTTL time.Duration,
) *BrandingService {
	return &BrandingService{
		repo:           repository,
		storage:        store,
		identityCache:  identityCache,
		logoPresignTTL: logoPresignTTL,
	}
}

func (s *BrandingService) Snapshot(ctx context.Context, customerID int) (auth.BrandingSnapshot, error) {
	row, err := s.repo.Find(ctx, customerID)
	if err != nil {
		return auth.BrandingSnapshot{}, brandingError(err)
	}

	snapshot := auth.BrandingSnapshot{
		Theme:    row.Theme,
		Language: row.Language,
		Timezone: row.Timezone,
	}

	if row.LogoKey == nil || *row.LogoKey == "" {
		return snapshot, nil
	}

	signed, err := s.storage.PresignGet(ctx, *row.LogoKey, s.logoPresignTTL)
	if err != nil {
		slog.WarnContext(ctx, "logo presign failed", "customer_id", customerID, "error", err)
		return snapshot, nil
	}

	snapshot.LogoURL = &signed

	return snapshot, nil
}

func (s *BrandingService) Update(ctx context.Context, customerID int, input BrandingInput) error {
	updates, err := preferenceUpdates(input)
	if err != nil {
		return err
	}

	if len(updates) == 0 {
		return nil
	}

	updates["updated_at"] = time.Now()

	affected, err := s.repo.Update(ctx, customerID, updates)
	if err != nil {
		return brandingError(err)
	}
	if affected == 0 {
		return utils.ErrBrandingNotFound
	}

	s.invalidate(ctx, customerID)

	return nil
}

func (s *BrandingService) UploadLogo(ctx context.Context, customerID int, header *multipart.FileHeader) error {
	payload, contentType, extension, err := readLogo(header)
	if err != nil {
		return err
	}

	current, err := s.repo.Find(ctx, customerID)
	if err != nil {
		return brandingError(err)
	}

	key := logoKey(customerID, extension)

	err = s.storage.Put(ctx, key, bytes.NewReader(payload), int64(len(payload)), contentType)
	if err != nil {
		slog.ErrorContext(ctx, "logo upload failed", "customer_id", customerID, "error", err)
		return utils.ErrUnavailable
	}

	affected, err := s.repo.Update(ctx, customerID, map[string]any{
		"logo_key":   key,
		"updated_at": time.Now(),
	})
	if err != nil {
		return brandingError(err)
	}
	if affected == 0 {
		return utils.ErrBrandingNotFound
	}

	s.invalidate(ctx, customerID)
	s.removeStale(ctx, current.LogoKey, key)

	return nil
}

func (s *BrandingService) RemoveLogo(ctx context.Context, customerID int) error {
	current, err := s.repo.Find(ctx, customerID)
	if err != nil {
		return brandingError(err)
	}

	if current.LogoKey == nil || *current.LogoKey == "" {
		return nil
	}

	affected, err := s.repo.Update(ctx, customerID, map[string]any{
		"logo_key":   nil,
		"updated_at": time.Now(),
	})
	if err != nil {
		return brandingError(err)
	}
	if affected == 0 {
		return utils.ErrBrandingNotFound
	}

	s.invalidate(ctx, customerID)
	s.removeStale(ctx, current.LogoKey, "")

	return nil
}

func (s *BrandingService) invalidate(ctx context.Context, customerID int) {
	if err := s.identityCache.DropCustomerIdentities(ctx, customerID); err != nil {
		slog.WarnContext(ctx, "branding cache invalidation failed", "customer_id", customerID, "error", err)
	}
}

func (s *BrandingService) removeStale(ctx context.Context, previous *string, key string) {
	if previous == nil || *previous == "" || *previous == key {
		return
	}

	if err := s.storage.Remove(ctx, *previous); err != nil {
		slog.WarnContext(ctx, "stale logo removal failed", "key", *previous, "error", err)
	}
}

func preferenceUpdates(input BrandingInput) (map[string]any, error) {
	updates := make(map[string]any, 3)

	if input.Theme != nil {
		theme := strings.ToUpper(strings.TrimSpace(*input.Theme))
		if theme != db.ThemeLight && theme != db.ThemeDark {
			return nil, utils.ErrInvalidTheme
		}

		updates["theme"] = theme
	}

	if input.Language != nil {
		language := strings.ToUpper(strings.TrimSpace(*input.Language))

		switch language {
		case db.LanguageEnglish, db.LanguageJapanese, db.LanguageSpanish:
			updates["language"] = language
		default:
			return nil, utils.ErrInvalidLanguage
		}
	}

	if input.Timezone != nil {
		timezone, err := normalizeTimezone(*input.Timezone)
		if err != nil {
			return nil, err
		}

		updates["timezone"] = timezone
	}

	return updates, nil
}

func normalizeTimezone(value string) (string, error) {
	timezone := strings.TrimSpace(value)

	if timezone == "" || timezone == "Local" {
		return "", utils.ErrInvalidTimezone
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return "", utils.ErrInvalidTimezone
	}

	return timezone, nil
}

func readLogo(header *multipart.FileHeader) ([]byte, string, string, error) {
	if header == nil || header.Size <= 0 || header.Size > MaxLogoSize {
		return nil, "", "", utils.ErrInvalidLogo
	}

	opened, err := header.Open()
	if err != nil {
		return nil, "", "", utils.ErrInvalidLogo
	}
	defer opened.Close()

	payload, err := io.ReadAll(io.LimitReader(opened, MaxLogoSize+1))
	if err != nil {
		return nil, "", "", utils.ErrInvalidLogo
	}

	if len(payload) == 0 || len(payload) > MaxLogoSize {
		return nil, "", "", utils.ErrInvalidLogo
	}

	contentType, extension, ok := detectLogoType(payload)
	if !ok {
		return nil, "", "", utils.ErrInvalidLogo
	}

	if err := verifyImage(payload); err != nil {
		return nil, "", "", err
	}

	return payload, contentType, extension, nil
}

func detectLogoType(payload []byte) (string, string, bool) {
	switch http.DetectContentType(payload) {
	case "image/png":
		return "image/png", ".png", true
	case "image/jpeg":
		return "image/jpeg", ".jpg", true
	case "image/webp":
		return "image/webp", ".webp", true
	}

	return "", "", false
}

func verifyImage(payload []byte) error {
	config, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil {
		return utils.ErrInvalidLogo
	}

	if config.Width <= 0 || config.Height <= 0 {
		return utils.ErrInvalidLogo
	}

	if config.Width > maxLogoDimension || config.Height > maxLogoDimension {
		return utils.ErrInvalidLogo
	}

	if _, _, err := image.Decode(bytes.NewReader(payload)); err != nil {
		return utils.ErrInvalidLogo
	}

	return nil
}

func logoKey(customerID int, extension string) string {
	return "customers/" + strconv.Itoa(customerID) + "/branding/logo/logo" + extension
}

func brandingError(err error) error {
	if db.IsNotFound(err) {
		return utils.ErrBrandingNotFound
	}

	return err
}
