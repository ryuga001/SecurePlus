package email

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"
)

type AuthorizedDomain struct {
	ConfigID   int `json:"config_id"`
	CustomerID int `json:"customer_id"`
}

type DomainRegistry interface {
	SyncDomain(ctx context.Context, previous, domain string, value AuthorizedDomain) error
	DropDomain(ctx context.Context, domain string) error
}

type RedisService struct {
	rdb *redis.Client
}

func NewRedisService(rdb *redis.Client) *RedisService {
	return &RedisService{rdb: rdb}
}

func authorizedDomainKey(domain string) string { return "authorized_domain:" + domain }

func staleKey(previous, domain string) string {
	if previous == "" || previous == domain {
		return ""
	}

	return authorizedDomainKey(previous)
}

func (s *RedisService) SyncDomain(ctx context.Context, previous, domain string, value AuthorizedDomain) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	pipe := s.rdb.TxPipeline()

	if stale := staleKey(previous, domain); stale != "" {
		pipe.Del(ctx, stale)
	}

	pipe.Set(ctx, authorizedDomainKey(domain), payload, 0)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (s *RedisService) DropDomain(ctx context.Context, domain string) error {
	return s.rdb.Del(ctx, authorizedDomainKey(domain)).Err()
}

func (s *RedisService) Domain(ctx context.Context, domain string) (AuthorizedDomain, error) {
	raw, err := s.rdb.Get(ctx, authorizedDomainKey(domain)).Bytes()
	if errors.Is(err, redis.Nil) {
		return AuthorizedDomain{}, ErrNotFound
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
