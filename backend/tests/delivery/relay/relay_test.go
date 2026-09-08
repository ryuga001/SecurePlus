package relay_test

import (
	"context"
	"errors"
	"net"
	"net/textproto"
	"testing"
	"time"

	"dpdp-backend/internal/delivery/relay"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		code      int
		permanent bool
	}{
		{"nil", nil, 0, false},
		{"transient smtp", &textproto.Error{Code: 451, Msg: "try later"}, 451, false},
		{"permanent smtp", &textproto.Error{Code: 550, Msg: "no such user"}, 550, true},
		{"mailbox full", &textproto.Error{Code: 452, Msg: "over quota"}, 452, false},
		{"dial timeout", errors.New("dial tcp: i/o timeout"), 0, false},
		{"no destination", relay.ErrNoDestination, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, permanent := relay.Classify(tc.err)
			if code != tc.code || permanent != tc.permanent {
				t.Fatalf("Classify = (%d, %v), want (%d, %v)", code, permanent, tc.code, tc.permanent)
			}
		})
	}
}

func TestNextBackoff(t *testing.T) {
	first := time.Minute
	second := relay.NextBackoff(first, 5, 15*time.Minute)

	if second != 5*time.Minute {
		t.Fatalf("second delay = %v, want 5m", second)
	}

	third := relay.NextBackoff(second, 5, 15*time.Minute)
	if third != 15*time.Minute {
		t.Fatalf("third delay = %v, want the 15m cap", third)
	}

	if capped := relay.NextBackoff(third, 5, 15*time.Minute); capped != 15*time.Minute {
		t.Fatalf("delay grew past the cap: %v", capped)
	}

	if zero := relay.NextBackoff(time.Minute, 0, time.Hour); zero != time.Minute {
		t.Fatalf("multiplier below one changed the delay: %v", zero)
	}
}

func TestResolverHonoursOverride(t *testing.T) {
	resolver := relay.NewResolver(time.Second, false, "mailpit:1025")

	destinations, err := resolver.Destinations(context.Background(), "anything.test")
	if err != nil {
		t.Fatalf("Destinations returned %v", err)
	}

	if len(destinations) != 1 {
		t.Fatalf("expected one destination, got %d", len(destinations))
	}
	if destinations[0].Host != "mailpit" || destinations[0].Addr != net.JoinHostPort("mailpit", "1025") {
		t.Fatalf("destination = %+v", destinations[0])
	}
}

func TestResolverOverrideWithoutPortUses25(t *testing.T) {
	resolver := relay.NewResolver(time.Second, false, "relay.internal")

	destinations, err := resolver.Destinations(context.Background(), "anything.test")
	if err != nil {
		t.Fatalf("Destinations returned %v", err)
	}

	if destinations[0].Addr != net.JoinHostPort("relay.internal", "25") {
		t.Fatalf("addr = %q", destinations[0].Addr)
	}
}

func TestResolverUnknownDomain(t *testing.T) {
	resolver := relay.NewResolver(2*time.Second, false, "")

	if _, err := resolver.Destinations(context.Background(), "invalid.invalid"); err == nil {
		t.Fatal("expected a resolution failure for an invalid TLD")
	}
}
