package emailprovider

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"

	"dpdp-backend/internal/admin/utils"
)

type AuthorizedDomain struct {
	ConfigID   int `json:"config_id"`
	CustomerID int `json:"customer_id"`
}

type DomainRegistry interface {
	SyncDomain(ctx context.Context, previous, domain string, value AuthorizedDomain) error
	DropDomain(ctx context.Context, domain string) error
}

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{rdb: rdb}
}

func AuthorizedDomainKey(domain string) string { return "authorized_domain:" + domain }

func StaleKey(previous, domain string) string {
	if previous == "" || previous == domain {
		return ""
	}

	return AuthorizedDomainKey(previous)
}

func (r *RedisRepository) SyncDomain(ctx context.Context, previous, domain string, value AuthorizedDomain) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	pipe := r.rdb.TxPipeline()

	if stale := StaleKey(previous, domain); stale != "" {
		pipe.Del(ctx, stale)
	}

	pipe.Set(ctx, AuthorizedDomainKey(domain), payload, 0)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *RedisRepository) DropDomain(ctx context.Context, domain string) error {
	return r.rdb.Del(ctx, AuthorizedDomainKey(domain)).Err()
}

func (r *RedisRepository) Domain(ctx context.Context, domain string) (AuthorizedDomain, error) {
	raw, err := r.rdb.Get(ctx, AuthorizedDomainKey(domain)).Bytes()
	if errors.Is(err, redis.Nil) {
		return AuthorizedDomain{}, utils.ErrConfigurationNotFound
	}
	if err != nil {
		return AuthorizedDomain{}, err
	}

	var value AuthorizedDomain
	if err := json.Unmarshal(raw, &value); err != nil {
		return AuthorizedDomain{}, err
	}

	return value, nil
}
