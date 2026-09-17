package config

import "time"

type Delivery struct {
	Stream        string
	ConsumerGroup string
	Workers       int
	ClaimIdle     time.Duration
	BatchSize     int64
	BlockTime     time.Duration
}

func loadDelivery() Delivery {
	return Delivery{
		Stream:        envString("DELIVERY_STREAM", "delivery:messages"),
		ConsumerGroup: envString("DELIVERY_CONSUMER_GROUP", "delivery-workers"),
		Workers:       envInt("DELIVERY_WORKERS", 4),
		ClaimIdle:     envDuration("DELIVERY_CLAIM_IDLE", 5*time.Minute),
		BatchSize:     int64(envInt("DELIVERY_BATCH_SIZE", 10)),
		BlockTime:     envDuration("DELIVERY_BLOCK_TIME", time.Second),
	}
}
