package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"dpdp-backend/internal/config"
)

type Queued struct {
	Entry   string
	Message EmailMessage
}

type Queue struct {
	rdb    *redis.Client
	stream string
	group  string

	published atomic.Int64
	failed    atomic.Int64
}

func NewQueue(rdb *redis.Client, cfg config.Delivery) *Queue {
	return &Queue{rdb: rdb, stream: cfg.Stream, group: cfg.ConsumerGroup}
}

func (q *Queue) EnsureGroup(ctx context.Context) error {
	err := q.rdb.XGroupCreateMkStream(ctx, q.stream, q.group, "$").Err()
	if err == nil || strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return nil
	}

	return err
}

func (q *Queue) Publish(ctx context.Context, msg EmailMessage) error {
	recipients, err := json.Marshal(msg.Recipients)
	if err != nil {
		q.failed.Add(1)
		return err
	}

	err = q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]any{
			"id":             msg.MessageID,
			"correlation_id": msg.CorrelationID,
			"customer_id":    msg.CustomerID,
			"config_id":      msg.ConfigID,
			"envelope_from":  msg.From,
			"sender_domain":  msg.SenderDomain,
			"recipients":     string(recipients),
			"received_at":    msg.ReceivedAt.Format(time.RFC3339Nano),
			"raw":            msg.Raw,
		},
	}).Err()
	if err != nil {
		q.failed.Add(1)
		return err
	}

	q.published.Add(1)

	return nil
}

func (q *Queue) Consume(ctx context.Context, consumer string, count int64, block time.Duration) ([]Queued, error) {
	streams, err := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.group,
		Consumer: consumer,
		Streams:  []string{q.stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}

		return nil, err
	}

	return decodeStreams(streams), nil
}

func (q *Queue) Reclaim(ctx context.Context, consumer string, minIdle time.Duration, count int64) ([]Queued, error) {
	messages, _, err := q.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   q.stream,
		Group:    q.group,
		Consumer: consumer,
		MinIdle:  minIdle,
		Start:    "0-0",
		Count:    count,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}

		return nil, err
	}

	return decodeMessages(messages), nil
}

func (q *Queue) Ack(ctx context.Context, entry string) error {
	return q.rdb.XAck(ctx, q.stream, q.group, entry).Err()
}

func (q *Queue) Pending(ctx context.Context) (int64, error) {
	pending, err := q.rdb.XPending(ctx, q.stream, q.group).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}

		return 0, err
	}

	return pending.Count, nil
}

func (q *Queue) Stats() (published, failed int64) {
	return q.published.Load(), q.failed.Load()
}

func decodeStreams(streams []redis.XStream) []Queued {
	queued := make([]Queued, 0)

	for _, stream := range streams {
		queued = append(queued, decodeMessages(stream.Messages)...)
	}

	return queued
}

func decodeMessages(messages []redis.XMessage) []Queued {
	queued := make([]Queued, 0, len(messages))

	for _, message := range messages {
		queued = append(queued, Queued{Entry: message.ID, Message: decodeMessage(message.Values)})
	}

	return queued
}

func decodeMessage(values map[string]any) EmailMessage {
	raw := []byte(field(values, "raw"))

	msg := EmailMessage{
		MessageID:     field(values, "id"),
		CorrelationID: field(values, "correlation_id"),
		CustomerID:    number(values, "customer_id"),
		ConfigID:      number(values, "config_id"),
		From:          field(values, "envelope_from"),
		SenderDomain:  field(values, "sender_domain"),
		Raw:           raw,
		Size:          int64(len(raw)),
	}

	if recipients := field(values, "recipients"); recipients != "" {
		json.Unmarshal([]byte(recipients), &msg.Recipients)
	}

	if received, err := time.Parse(time.RFC3339Nano, field(values, "received_at")); err == nil {
		msg.ReceivedAt = received
	}

	return msg
}

func field(values map[string]any, key string) string {
	value, ok := values[key].(string)
	if !ok {
		return ""
	}

	return value
}

func number(values map[string]any, key string) int {
	parsed, err := strconv.Atoi(field(values, key))
	if err != nil {
		return 0
	}

	return parsed
}
