package engine_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/engine"
)

type fakeLoader struct {
	calls   atomic.Int32
	release chan struct{}
	cfg     delivery.TenantConfig
	err     error
}

func (f *fakeLoader) SigningConfig(_ context.Context, customerID, configID int) (delivery.TenantConfig, error) {
	f.calls.Add(1)

	if f.release != nil {
		<-f.release
	}

	if f.err != nil {
		return delivery.TenantConfig{}, f.err
	}

	cfg := f.cfg
	cfg.CustomerID = customerID
	cfg.ConfigID = configID

	return cfg, nil
}

func TestConfigKey(t *testing.T) {
	if got := engine.ConfigKey(12); got != "delivery:config:12" {
		t.Fatalf("ConfigKey = %q", got)
	}
}

func TestResolveAppliesDefaultSelector(t *testing.T) {
	loader := &fakeLoader{cfg: delivery.TenantConfig{Domain: "example.com", DKIMPrivateKey: "key"}}
	cache := engine.NewConfigCache(loader, nil)

	cfg, err := cache.Resolve(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}

	if cfg.DKIMSelector != delivery.DefaultDKIMSelector {
		t.Fatalf("selector = %q", cfg.DKIMSelector)
	}
	if cfg.CustomerID != 1 || cfg.ConfigID != 2 {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestResolvePropagatesLoaderFailure(t *testing.T) {
	failure := errors.New("no signing config")
	cache := engine.NewConfigCache(&fakeLoader{err: failure}, nil)

	if _, err := cache.Resolve(context.Background(), 1, 2); !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the loader failure", err)
	}
}

func TestResolveCollapsesConcurrentMisses(t *testing.T) {
	loader := &fakeLoader{
		release: make(chan struct{}),
		cfg:     delivery.TenantConfig{Domain: "example.com", DKIMPrivateKey: "key"},
	}
	cache := engine.NewConfigCache(loader, nil)

	const callers = 50

	var waiting sync.WaitGroup
	var done sync.WaitGroup

	waiting.Add(callers)
	done.Add(callers)

	for range callers {
		go func() {
			defer done.Done()

			waiting.Done()

			if _, err := cache.Resolve(context.Background(), 1, 2); err != nil {
				t.Error(err)
			}
		}()
	}

	waiting.Wait()
	close(loader.release)
	done.Wait()

	if calls := loader.calls.Load(); calls > 2 {
		t.Fatalf("loader called %d times, singleflight should collapse concurrent misses", calls)
	}
}

func TestProcessIsPassThrough(t *testing.T) {
	loader := &fakeLoader{cfg: delivery.TenantConfig{Domain: "example.com", DKIMPrivateKey: "key"}}
	processor := engine.NewEngine(engine.NewConfigCache(loader, nil))

	original := delivery.EmailMessage{
		CorrelationID: "abc",
		CustomerID:    1,
		ConfigID:      2,
		From:          "alice@example.com",
		Recipients:    []string{"bob@other.test"},
		Raw:           []byte("From: alice@example.com\r\n\r\nbody\r\n"),
	}

	processed, cfg, err := processor.Process(context.Background(), original)
	if err != nil {
		t.Fatalf("Process returned %v", err)
	}

	if string(processed.Raw) != string(original.Raw) {
		t.Fatal("the message body was modified")
	}
	if processed.CorrelationID != original.CorrelationID || processed.From != original.From {
		t.Fatalf("message = %+v", processed)
	}
	if cfg.Domain != "example.com" {
		t.Fatalf("config = %+v", cfg)
	}
}
