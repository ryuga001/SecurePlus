package config

import (
	"encoding/base64"
	"errors"
	"os"
	"time"
)

const (
	masterKeyLength = 32

	minQueueCapacity    = 1
	maxQueueCapacity    = 5000
	minChunkBytes       = 64 << 10
	maxChunkBytes       = 8 << 20
	minEvaluatorWorkers = 1
	maxEvaluatorWorkers = 16
	minSpoolBytes       = 1 << 20
)

type DataDiscovery struct {
	MasterKey   string
	KeyVersion  int
	TestTimeout time.Duration

	ScannerEnabled   bool
	QueueCapacity    int
	ChunkBytes       int
	EvaluatorWorkers int
	FileTimeout      time.Duration
	SpoolDir         string
	MaxSpoolBytes    int64
}

func loadDataDiscovery() DataDiscovery {
	return DataDiscovery{
		MasterKey:   os.Getenv("DATA_DISCOVERY_MASTER_KEY"),
		KeyVersion:  envInt("DATA_DISCOVERY_KEY_VERSION", 1),
		TestTimeout: envDuration("DATA_DISCOVERY_TEST_TIMEOUT", 20*time.Second),

		ScannerEnabled:   envBool("DATA_DISCOVERY_SCANNER_ENABLED", true),
		QueueCapacity:    envInt("DATA_DISCOVERY_QUEUE_CAPACITY", 100),
		ChunkBytes:       envInt("DATA_DISCOVERY_CHUNK_BYTES", 1<<20),
		EvaluatorWorkers: envInt("DATA_DISCOVERY_EVALUATOR_WORKERS", 4),
		FileTimeout:      envDuration("DATA_DISCOVERY_FILE_TIMEOUT", 30*time.Minute),
		SpoolDir:         envString("DATA_DISCOVERY_SPOOL_DIR", os.TempDir()),
		MaxSpoolBytes:    int64(envInt("DATA_DISCOVERY_MAX_SPOOL_BYTES", 1<<30)),
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

	if d.QueueCapacity < minQueueCapacity || d.QueueCapacity > maxQueueCapacity {
		return errors.New("DATA_DISCOVERY_QUEUE_CAPACITY must be between 1 and 5000")
	}

	if d.ChunkBytes < minChunkBytes || d.ChunkBytes > maxChunkBytes {
		return errors.New("DATA_DISCOVERY_CHUNK_BYTES must be between 65536 and 8388608")
	}

	if d.EvaluatorWorkers < minEvaluatorWorkers || d.EvaluatorWorkers > maxEvaluatorWorkers {
		return errors.New("DATA_DISCOVERY_EVALUATOR_WORKERS must be between 1 and 16")
	}

	if d.FileTimeout <= 0 {
		return errors.New("DATA_DISCOVERY_FILE_TIMEOUT must be a positive duration")
	}

	if d.MaxSpoolBytes < minSpoolBytes {
		return errors.New("DATA_DISCOVERY_MAX_SPOOL_BYTES must be at least 1048576")
	}

	if info, err := os.Stat(d.SpoolDir); err != nil || !info.IsDir() {
		return errors.New("DATA_DISCOVERY_SPOOL_DIR must be an existing directory")
	}

	return nil
}
