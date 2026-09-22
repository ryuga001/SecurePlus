package config

import "time"

type Notification struct {
	Stream         string
	ConsumerGroup  string
	DeadStream     string
	Workers        int
	ClaimIdle      time.Duration
	BatchSize      int64
	BlockTime      time.Duration
	MaxDeliveries  int64
	PublishTimeout time.Duration
}

func loadNotification() Notification {
	stream := envString("NOTIFICATION_STREAM", "notification:messages")

	return Notification{
		Stream:         stream,
		ConsumerGroup:  envString("NOTIFICATION_CONSUMER_GROUP", "notification-workers"),
		DeadStream:     envString("NOTIFICATION_DEAD_STREAM", stream+":dead"),
		Workers:        envInt("NOTIFICATION_WORKERS", 1),
		ClaimIdle:      envDuration("NOTIFICATION_CLAIM_IDLE", 5*time.Minute),
		BatchSize:      int64(envInt("NOTIFICATION_BATCH_SIZE", 10)),
		BlockTime:      envDuration("NOTIFICATION_BLOCK_TIME", time.Second),
		MaxDeliveries:  int64(envInt("NOTIFICATION_MAX_DELIVERIES", 5)),
		PublishTimeout: envDuration("NOTIFICATION_PUBLISH_TIMEOUT", 10*time.Second),
	}
}
