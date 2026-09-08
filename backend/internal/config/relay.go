package config

import (
	"os"
	"time"
)

type Relay struct {
	HELOHost          string
	DialTimeout       time.Duration
	DNSTimeout        time.Duration
	MaxAttempts       int
	BackoffInitial    time.Duration
	BackoffMultiplier int
	BackoffMax        time.Duration
	TLSRequired       bool
	IPv6              bool
	MXOverride        string
}

func loadRelay() Relay {
	return Relay{
		HELOHost:          envString("RELAY_HELO_HOST", "dpdp.local"),
		DialTimeout:       envDuration("RELAY_DIAL_TIMEOUT", 30*time.Second),
		DNSTimeout:        envDuration("RELAY_DNS_TIMEOUT", 10*time.Second),
		MaxAttempts:       envInt("RELAY_MAX_ATTEMPTS", 3),
		BackoffInitial:    envDuration("RELAY_BACKOFF_INITIAL", time.Minute),
		BackoffMultiplier: envInt("RELAY_BACKOFF_MULTIPLIER", 5),
		BackoffMax:        envDuration("RELAY_BACKOFF_MAX", 15*time.Minute),
		TLSRequired:       envBool("RELAY_TLS_REQUIRED", false),
		IPv6:              envBool("RELAY_IPV6", false),
		MXOverride:        os.Getenv("RELAY_MX_OVERRIDE"),
	}
}
