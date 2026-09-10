package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	rowdto "dpdp-backend/internal/delivery/engine/policy/dto/policyset"
	"dpdp-backend/internal/delivery/engine/policy/services/cache"
	"dpdp-backend/internal/delivery/utils"
)

type countingStore struct {
	calls   atomic.Int32
	release chan struct{}
	rows    []rowdto.PolicyRuleRow
	err     error
}

func (s *countingStore) Resolve(_ context.Context, _ int, _, _ string) ([]rowdto.PolicyRuleRow, error) {
	s.calls.Add(1)

	if s.release != nil {
		<-s.release
	}

	if s.err != nil {
		return nil, s.err
	}

	return s.rows, nil
}

type countingBuilder struct {
	calls atomic.Int32
	err   error
}

func (b *countingBuilder) Build(set dto.PolicySet) (*dto.CompiledSet, error) {
	b.calls.Add(1)

	if b.err != nil {
		return nil, b.err
	}

	return &dto.CompiledSet{
		CustomerID:  set.CustomerID,
		EmailUserID: set.EmailUserID,
		PolicyCount: len(set.Policies),
	}, nil
}

func rows() []rowdto.PolicyRuleRow {
	ruleID := 5
	ruleName := "Cards"
	ruleType := utils.MatcherKeyword
	ruleValue := "card"

	return []rowdto.PolicyRuleRow{{
		PolicyID:                 1,
		PolicyName:               "Data Loss",
		Action:                   utils.ActionBlock,
		DomainRestrictionRaw:     []byte(`{"mode":"BLOCK","values":["blocked.test"]}`),
		AttachmentRestrictionRaw: []byte(`{"mode":"NONE","values":[]}`),
		EmailUserID:              42,
		RuleID:                   &ruleID,
		RuleName:                 &ruleName,
		RuleType:                 &ruleType,
		RuleValue:                &ruleValue,
	}}
}

func TestConcurrentMissesCollapseToOneQuery(t *testing.T) {
	store := &countingStore{release: make(chan struct{}), rows: rows()}
	builder := &countingBuilder{}
	service := cache.NewPolicyCacheService(store, builder, nil, time.Minute)

	const callers = 50

	var waiting sync.WaitGroup
	var done sync.WaitGroup

	waiting.Add(callers)
	done.Add(callers)

	for range callers {
		go func() {
			defer done.Done()

			waiting.Done()

			if _, err := service.Load(context.Background(), 1, "alice@example.com"); err != nil {
				t.Error(err)
			}
		}()
	}

	waiting.Wait()
	close(store.release)
	done.Wait()

	if calls := store.calls.Load(); calls != 1 {
		t.Fatalf("resolver called %d times, want exactly 1", calls)
	}
	if calls := builder.calls.Load(); calls != 1 {
		t.Fatalf("builder called %d times, want exactly 1", calls)
	}
}

func TestDifferentSendersDoNotCollapse(t *testing.T) {
	store := &countingStore{rows: rows()}
	service := cache.NewPolicyCacheService(store, &countingBuilder{}, nil, time.Minute)

	var done sync.WaitGroup

	for _, sender := range []string{"alice@example.com", "bob@example.com"} {
		done.Add(25)

		for range 25 {
			go func(address string) {
				defer done.Done()

				if _, err := service.Load(context.Background(), 1, address); err != nil {
					t.Error(err)
				}
			}(sender)
		}
	}

	done.Wait()

	if calls := store.calls.Load(); calls != 2 {
		t.Fatalf("resolver called %d times, want one per sender", calls)
	}
}

func TestWarmCacheDoesNotQueryAgain(t *testing.T) {
	store := &countingStore{rows: rows()}
	service := cache.NewPolicyCacheService(store, &countingBuilder{}, nil, time.Minute)

	for range 5 {
		if _, err := service.Load(context.Background(), 1, "alice@example.com"); err != nil {
			t.Fatalf("Load returned %v", err)
		}
	}

	if calls := store.calls.Load(); calls != 1 {
		t.Fatalf("resolver called %d times, want 1", calls)
	}
}

func TestExpiredEntryIsRebuilt(t *testing.T) {
	store := &countingStore{rows: rows()}
	service := cache.NewPolicyCacheService(store, &countingBuilder{}, nil, time.Nanosecond)

	if _, err := service.Load(context.Background(), 1, "alice@example.com"); err != nil {
		t.Fatalf("Load returned %v", err)
	}

	time.Sleep(time.Millisecond)

	if _, err := service.Load(context.Background(), 1, "alice@example.com"); err != nil {
		t.Fatalf("Load returned %v", err)
	}

	if calls := store.calls.Load(); calls != 2 {
		t.Fatalf("resolver called %d times, want a rebuild after expiry", calls)
	}
}

func TestResolverFailureIsSharedAndNotCached(t *testing.T) {
	failure := errors.New("postgres down")
	store := &countingStore{err: failure}
	service := cache.NewPolicyCacheService(store, &countingBuilder{}, nil, time.Minute)

	if _, err := service.Load(context.Background(), 1, "alice@example.com"); !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the resolver failure", err)
	}

	store.err = nil
	store.rows = rows()

	if _, err := service.Load(context.Background(), 1, "alice@example.com"); err != nil {
		t.Fatalf("a failure must not poison the cache: %v", err)
	}

	if calls := store.calls.Load(); calls != 2 {
		t.Fatalf("resolver called %d times, want a retry after failure", calls)
	}
}

func TestBuildFailurePropagates(t *testing.T) {
	failure := errors.New("too many rules")
	service := cache.NewPolicyCacheService(&countingStore{rows: rows()}, &countingBuilder{err: failure}, nil, time.Minute)

	if _, err := service.Load(context.Background(), 1, "alice@example.com"); !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the build failure", err)
	}
}

func TestSenderIsAssembledIntoAPolicySet(t *testing.T) {
	service := cache.NewPolicyCacheService(&countingStore{rows: rows()}, &countingBuilder{}, nil, time.Minute)

	set, err := service.Load(context.Background(), 1, "alice@example.com")
	if err != nil {
		t.Fatalf("Load returned %v", err)
	}

	if set.PolicyCount != 1 {
		t.Fatalf("policy count = %d, want 1", set.PolicyCount)
	}
	if set.EmailUserID != 42 {
		t.Fatalf("email user = %d, want 42", set.EmailUserID)
	}
}

func TestSetKeyIsCaseInsensitiveOnSender(t *testing.T) {
	if cache.SetKey(1, "Alice@Example.com") != cache.SetKey(1, "alice@example.com") {
		t.Fatal("cache key must not vary with sender casing")
	}
	if cache.SetKey(1, "alice@example.com") == cache.SetKey(2, "alice@example.com") {
		t.Fatal("cache key must be scoped per customer")
	}
}

type capturingBuilder struct {
	set dto.PolicySet
}

func (b *capturingBuilder) Build(set dto.PolicySet) (*dto.CompiledSet, error) {
	b.set = set

	return &dto.CompiledSet{CustomerID: set.CustomerID, EmailUserID: set.EmailUserID}, nil
}

func sharedRuleRows() []rowdto.PolicyRuleRow {
	ruleID := 5
	ruleName := "Cards"
	ruleType := utils.MatcherKeyword
	ruleValue := "card"
	none := []byte(`{"mode":"NONE","values":[]}`)

	row := func(policyID int, policyName, action string) rowdto.PolicyRuleRow {
		return rowdto.PolicyRuleRow{
			PolicyID:                 policyID,
			PolicyName:               policyName,
			Action:                   action,
			DomainRestrictionRaw:     none,
			AttachmentRestrictionRaw: none,
			EmailUserID:              42,
			RuleID:                   &ruleID,
			RuleName:                 &ruleName,
			RuleType:                 &ruleType,
			RuleValue:                &ruleValue,
		}
	}

	return []rowdto.PolicyRuleRow{
		row(1, "Audit Only", utils.ActionAudit),
		row(2, "Hard Block", utils.ActionBlock),
	}
}

func assembled(t *testing.T, rows []rowdto.PolicyRuleRow) dto.PolicySet {
	t.Helper()

	builder := &capturingBuilder{}

	service := cache.NewPolicyCacheService(&countingStore{rows: rows}, builder, nil, time.Minute)

	if _, err := service.Load(context.Background(), 1, "alice@example.com"); err != nil {
		t.Fatalf("Load returned %v", err)
	}

	return builder.set
}

func TestOneRuleSharedByTwoPoliciesIsKeptForBoth(t *testing.T) {
	set := assembled(t, sharedRuleRows())

	if len(set.Policies) != 2 {
		t.Fatalf("policies = %+v, want both", set.Policies)
	}

	if len(set.Rules) != 2 {
		t.Fatalf("rules = %+v, one rule shared by two policies must appear once per policy", set.Rules)
	}

	actions := map[string]bool{}
	for _, rule := range set.Rules {
		if rule.RuleID != 5 {
			t.Fatalf("rule = %+v", rule)
		}

		actions[rule.Action] = true
	}

	if !actions[utils.ActionAudit] || !actions[utils.ActionBlock] {
		t.Fatalf("actions = %v, the blocking policy's action was dropped", actions)
	}
}

func TestDuplicateRowsForOnePolicyAndRuleCollapse(t *testing.T) {
	duplicated := append(sharedRuleRows()[:1], sharedRuleRows()[0])

	set := assembled(t, duplicated)

	if len(set.Policies) != 1 || len(set.Rules) != 1 {
		t.Fatalf("policies = %d rules = %d, want one of each", len(set.Policies), len(set.Rules))
	}
}
