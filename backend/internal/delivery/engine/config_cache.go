package engine

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"dpdp-backend/internal/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

const configCacheTTL = 10 * time.Minute

type SigningLoader interface {
	SigningConfig(ctx context.Context, customerID, configID int) (delivery.TenantConfig, error)
}

type ConfigCache struct {
	loader SigningLoader
	rdb    *redis.Client
	group  singleflight.Group
}

func NewConfigCache(loader SigningLoader, rdb *redis.Client) *ConfigCache {
	return &ConfigCache{loader: loader, rdb: rdb}
}

func ConfigKey(configID int) string {
	return "delivery:config:" + strconv.Itoa(configID)
}

func (c *ConfigCache) Resolve(ctx context.Context, customerID, configID int) (delivery.TenantConfig, error) {
	if cached, ok := c.read(ctx, configID); ok && cached.CustomerID == customerID {
		return cached, nil
	}

	value, err, _ := c.group.Do(ConfigKey(configID), func() (any, error) {
		cfg, err := c.loader.SigningConfig(ctx, customerID, configID)
		if err != nil {
			return delivery.TenantConfig{}, err
		}

		if cfg.DKIMSelector == "" {
			cfg.DKIMSelector = deliveryutils.DefaultDKIMSelector
		}

		c.write(ctx, cfg)

		return cfg, nil
	})

	if err != nil {
		return delivery.TenantConfig{}, err
	}

	cfg, ok := value.(delivery.TenantConfig)
	if !ok {
		return delivery.TenantConfig{}, deliveryutils.ErrSigningConfigMissing
	}

	return cfg, nil
}

func (c *ConfigCache) read(ctx context.Context, configID int) (delivery.TenantConfig, bool) {
	if c.rdb == nil {
		return delivery.TenantConfig{}, false
	}

	raw, err := c.rdb.Get(ctx, ConfigKey(configID)).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.WarnContext(ctx, "delivery config cache unavailable", "config_id", configID, "error", err)
		}

		return delivery.TenantConfig{}, false
	}

	var cfg delivery.TenantConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return delivery.TenantConfig{}, false
	}

	return cfg, true
}

func (c *ConfigCache) write(ctx context.Context, cfg delivery.TenantConfig) {
	if c.rdb == nil {
		return
	}

	payload, err := json.Marshal(cfg)
	if err != nil {
		return
	}

	if err := c.rdb.Set(ctx, ConfigKey(cfg.ConfigID), payload, configCacheTTL).Err(); err != nil {
		slog.WarnContext(ctx, "delivery config cache write failed", "config_id", cfg.ConfigID, "error", err)
	}
}
