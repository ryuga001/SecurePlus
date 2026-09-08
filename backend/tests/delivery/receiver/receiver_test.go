package receiver_test

import (
	"context"
	"errors"
	"testing"

	"dpdp-backend/internal/delivery/receiver"
)

type fakeCache struct {
	customerID int
	configID   int
	err        error
	remembered []string
}

func (f *fakeCache) Domain(_ context.Context, _ string) (int, int, error) {
	if f.err != nil {
		return 0, 0, f.err
	}

	return f.customerID, f.configID, nil
}

func (f *fakeCache) Remember(_ context.Context, domain string, _, _ int) error {
	f.remembered = append(f.remembered, domain)

	return nil
}

type fakeStore struct {
	customerID int
	configID   int
	err        error
	calls      int
}

func (f *fakeStore) FindByDomain(_ context.Context, _ string) (int, int, error) {
	f.calls++

	if f.err != nil {
		return 0, 0, f.err
	}

	return f.customerID, f.configID, nil
}

func TestMessageID(t *testing.T) {
	raw := []byte("From: alice@example.com\r\nMessage-ID: <abc@example.com>\r\nSubject: hi\r\n\r\nbody\r\n")

	if got := receiver.MessageID(raw); got != "<abc@example.com>" {
		t.Fatalf("MessageID = %q", got)
	}
}

func TestMessageIDMissing(t *testing.T) {
	raw := []byte("From: alice@example.com\r\nSubject: hi\r\n\r\nbody\r\n")

	if got := receiver.MessageID(raw); got != "" {
		t.Fatalf("MessageID = %q, want empty", got)
	}
}

func TestMessageIDMalformed(t *testing.T) {
	if got := receiver.MessageID([]byte("not a message at all")); got != "" {
		t.Fatalf("MessageID = %q, want empty", got)
	}
}

func TestAuthorizeFromCache(t *testing.T) {
	cache := &fakeCache{customerID: 7, configID: 3}
	store := &fakeStore{}

	authorization, err := receiver.NewAuthorizer(cache, store).Authorize(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Authorize returned %v", err)
	}

	if authorization.CustomerID != 7 || authorization.ConfigID != 3 {
		t.Fatalf("authorization = %+v", authorization)
	}
	if store.calls != 0 {
		t.Fatal("a cache hit must not reach the database")
	}
}

func TestAuthorizeFallsBackToStore(t *testing.T) {
	cache := &fakeCache{err: errors.New("redis down")}
	store := &fakeStore{customerID: 4, configID: 9}

	authorization, err := receiver.NewAuthorizer(cache, store).Authorize(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Authorize returned %v", err)
	}

	if authorization.CustomerID != 4 || authorization.ConfigID != 9 {
		t.Fatalf("authorization = %+v", authorization)
	}
	if store.calls != 1 {
		t.Fatalf("store calls = %d, want 1", store.calls)
	}
	if len(cache.remembered) != 1 || cache.remembered[0] != "example.com" {
		t.Fatalf("cache was not repopulated: %v", cache.remembered)
	}
}

func TestAuthorizeRejectsUnknownDomain(t *testing.T) {
	cache := &fakeCache{err: errors.New("miss")}
	store := &fakeStore{err: receiver.ErrDomainUnknown}

	_, err := receiver.NewAuthorizer(cache, store).Authorize(context.Background(), "stranger.test")
	if !errors.Is(err, receiver.ErrDomainUnknown) {
		t.Fatalf("error = %v, want ErrDomainUnknown", err)
	}
}

func TestAuthorizeSurfacesStoreFailure(t *testing.T) {
	failure := errors.New("postgres down")
	cache := &fakeCache{err: errors.New("redis down")}
	store := &fakeStore{err: failure}

	_, err := receiver.NewAuthorizer(cache, store).Authorize(context.Background(), "example.com")
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the store failure", err)
	}
	if errors.Is(err, receiver.ErrDomainUnknown) {
		t.Fatal("an outage must not be reported as an unknown domain")
	}
}
