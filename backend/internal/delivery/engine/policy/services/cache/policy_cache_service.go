package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"dpdp-backend/internal/db"
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	rowdto "dpdp-backend/internal/delivery/engine/policy/dto/policyset"
	"dpdp-backend/internal/delivery/utils"
)

type PolicySetStore interface {
	Resolve(ctx context.Context, customerID int, email, policyType string) ([]rowdto.PolicyRuleRow, error)
}

type Builder interface {
	Build(set dto.PolicySet) (*dto.CompiledSet, error)
}

type entry struct {
	set       *dto.CompiledSet
	expiresAt time.Time
}

type PolicyCacheService struct {
	store   PolicySetStore
	builder Builder
	rdb     *redis.Client
	ttl     time.Duration

	mu      sync.RWMutex
	entries map[string]entry
	group   singleflight.Group
}

func NewPolicyCacheService(store PolicySetStore, builder Builder, rdb *redis.Client, ttl time.Duration) *PolicyCacheService {
	if ttl <= 0 {
		ttl = utils.CacheTTL
	}

	return &PolicyCacheService{
		store:   store,
		builder: builder,
		rdb:     rdb,
		ttl:     ttl,
		entries: map[string]entry{},
	}
}

func SetKey(customerID int, sender string) string {
	return "policy:set:" + strconv.Itoa(customerID) + ":" + strings.ToLower(sender)
}

func (c *PolicyCacheService) Load(ctx context.Context, customerID int, sender string) (*dto.CompiledSet, error) {
	key := SetKey(customerID, sender)

	if cached, ok := c.lookup(key); ok {
		slog.DebugContext(ctx, "policy set cache hit", "customer_id", customerID)
		return cached, nil
	}

	value, err, _ := c.group.Do(key, func() (any, error) {
		if cached, ok := c.lookup(key); ok {
			return cached, nil
		}

		rows, err := c.rows(ctx, key, customerID, sender)
		if err != nil {
			return nil, err
		}

		compiled, err := c.builder.Build(assemble(customerID, rows))
		if err != nil {
			return nil, err
		}

		c.remember(key, compiled)

		return compiled, nil
	})
	if err != nil {
		return nil, err
	}

	compiled, ok := value.(*dto.CompiledSet)
	if !ok {
		return nil, utils.ErrSigningConfigMissing
	}

	return compiled, nil
}

func (c *PolicyCacheService) rows(ctx context.Context, key string, customerID int, sender string) ([]rowdto.PolicyRuleRow, error) {
	if cached, ok := c.readRedis(ctx, key); ok {
		return cached, nil
	}

	rows, err := c.store.Resolve(ctx, customerID, sender, db.PolicyTypeEmail)
	if err != nil {
		return nil, err
	}

	c.writeRedis(ctx, key, rows)

	return rows, nil
}

func (c *PolicyCacheService) lookup(key string) (*dto.CompiledSet, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	found, ok := c.entries[key]
	if !ok || time.Now().After(found.expiresAt) {
		return nil, false
	}

	return found.set, true
}

func (c *PolicyCacheService) remember(key string, set *dto.CompiledSet) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = entry{set: set, expiresAt: time.Now().Add(c.ttl)}
}

func (c *PolicyCacheService) readRedis(ctx context.Context, key string) ([]rowdto.PolicyRuleRow, bool) {
	if c.rdb == nil {
		return nil, false
	}

	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.WarnContext(ctx, "policy set cache unavailable", "error", err)
		}

		return nil, false
	}

	var rows []rowdto.PolicyRuleRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, false
	}

	return rows, true
}

func (c *PolicyCacheService) writeRedis(ctx context.Context, key string, rows []rowdto.PolicyRuleRow) {
	if c.rdb == nil {
		return
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		return
	}

	if err := c.rdb.Set(ctx, key, payload, c.ttl).Err(); err != nil {
		slog.WarnContext(ctx, "policy set cache write failed", "error", err)
	}
}

func assemble(customerID int, rows []rowdto.PolicyRuleRow) dto.PolicySet {
	set := dto.PolicySet{CustomerID: customerID}
	seen := map[int]bool{}
	rules := map[int]bool{}

	for _, row := range rows {
		if set.EmailUserID == 0 {
			set.EmailUserID = row.EmailUserID
		}

		if !seen[row.PolicyID] {
			seen[row.PolicyID] = true

			set.Policies = append(set.Policies, dto.PolicyRecord{
				PolicyID:              row.PolicyID,
				PolicyName:            row.PolicyName,
				Action:                row.Action,
				DomainRestriction:     decode(row.DomainRestrictionRaw),
				AttachmentRestriction: decode(row.AttachmentRestrictionRaw),
			})
		}

		if row.RuleID == nil || rules[*row.RuleID] {
			continue
		}

		rules[*row.RuleID] = true

		set.Rules = append(set.Rules, dto.RuleRecord{
			PolicyID:   row.PolicyID,
			PolicyName: row.PolicyName,
			Action:     row.Action,
			RuleID:     *row.RuleID,
			RuleName:   text(row.RuleName),
			RuleType:   text(row.RuleType),
			RuleValue:  text(row.RuleValue),
		})
	}

	return set
}

func decode(raw []byte) dto.Restriction {
	restriction := dto.Restriction{Mode: utils.RestrictionNone, Values: []string{}}

	if len(raw) == 0 {
		return restriction
	}

	if err := json.Unmarshal(raw, &restriction); err != nil {
		return dto.Restriction{Mode: utils.RestrictionNone, Values: []string{}}
	}

	if restriction.Mode == "" {
		restriction.Mode = utils.RestrictionNone
	}

	if restriction.Values == nil {
		restriction.Values = []string{}
	}

	return restriction
}

func text(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
