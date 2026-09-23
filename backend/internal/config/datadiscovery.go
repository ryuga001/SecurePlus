package config

import (
	"encoding/base64"
	"errors"
	"os"
	"time"
)

const masterKeyLength = 32

type DataDiscovery struct {
	MasterKey   string
	KeyVersion  int
	TestTimeout time.Duration
}

func loadDataDiscovery() DataDiscovery {
	return DataDiscovery{
		MasterKey:   os.Getenv("DATA_DISCOVERY_MASTER_KEY"),
		KeyVersion:  envInt("DATA_DISCOVERY_KEY_VERSION", 1),
		TestTimeout: envDuration("DATA_DISCOVERY_TEST_TIMEOUT", 20*time.Second),
	}
}

func (d DataDiscovery) Validate() error {
	decoded, err := base64.StdEncoding.DecodeString(d.MasterKey)
	if err != nil || len(decoded) != masterKeyLength {
		return errors.New("DATA_DISCOVERY_MASTER_KEY must be 32 random bytes, base64 encoded (openssl rand -base64 32)")
	}

	if d.KeyVersion < 1 {
		return errors.New("DATA_DISCOVERY_KEY_VERSION must be 1 or greater")
	}

	if d.TestTimeout <= 0 {
		return errors.New("DATA_DISCOVERY_TEST_TIMEOUT must be a positive duration")
	}

	return nil
}
