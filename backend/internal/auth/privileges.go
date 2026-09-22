package auth

import (
	"context"
	"log/slog"

	"gorm.io/gorm"

	"dpdp-backend/internal/config"
	"dpdp-backend/internal/db"
)

func PrivilegeNames(ctx context.Context, database *gorm.DB, store *Store, roleID int) ([]string, error) {
	cached, err := store.Privileges(ctx, roleID)
	if err == nil && len(cached) > 0 {
		return withoutBlanks(cached), nil
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

	return withoutBlanks(names), nil
}

func withoutBlanks(names []string) []string {
	cleaned := make([]string, 0, len(names))

	for _, name := range names {
		if name != "" {
			cleaned = append(cleaned, name)
		}
	}

	return cleaned
}

func (s *Service) Privileges(ctx context.Context, roleID int) ([]string, error) {
	return PrivilegeNames(ctx, s.db, s.store, roleID)
}
