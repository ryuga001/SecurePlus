package worker

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
	"dpdp-backend/internal/notification"
)

type Queued struct {
	Entry   string
	Message notification.NotificationMessage
	Retries int64
}

type NotificationQueue struct {
	rdb    *redis.Client
	stream string
	group  string
	dead   string

	published atomic.Int64
	failed    atomic.Int64
}

func NewNotificationQueue(rdb *redis.Client, cfg config.Notification) *NotificationQueue {
	return &NotificationQueue{
		rdb:    rdb,
		stream: cfg.Stream,
		group:  cfg.ConsumerGroup,
		dead:   cfg.DeadStream,
	}
}

func (q *NotificationQueue) EnsureGroup(ctx context.Context) error {
	err := q.rdb.XGroupCreateMkStream(ctx, q.stream, q.group, "$").Err()
	if err == nil || strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return nil
	}

	return err
}

func (q *NotificationQueue) Publish(ctx context.Context, msg notification.NotificationMessage) error {
	values, err := encodeNotificationMessage(msg)
	if err != nil {
		q.failed.Add(1)
		return err
	}

	err = q.rdb.XAdd(ctx, &redis.XAddArgs{Stream: q.stream, Values: values}).Err()
	if err != nil {
		q.failed.Add(1)
		return err
	}

	q.published.Add(1)

	return nil
}

func (q *NotificationQueue) Consume(
	ctx context.Context,
	consumer string,
	count int64,
	block time.Duration,
) ([]Queued, error) {
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

	queued := make([]Queued, 0)
	for _, stream := range streams {
		queued = append(queued, decodeMessages(stream.Messages)...)
	}

	return queued, nil
}

func (q *NotificationQueue) Reclaim(
	ctx context.Context,
	consumer string,
	minIdle time.Duration,
	count int64,
) ([]Queued, error) {
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

	queued := decodeMessages(messages)
	if len(queued) == 0 {
		return queued, nil
	}

	deliveries, err := q.deliveryCounts(ctx, count)
	if err != nil {
		return queued, nil
	}

	for index := range queued {
		queued[index].Retries = deliveries[queued[index].Entry]
	}

	return queued, nil
}

func (q *NotificationQueue) deliveryCounts(ctx context.Context, count int64) (map[string]int64, error) {
	pending, err := q.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.stream,
		Group:  q.group,
		Start:  "-",
		End:    "+",
		Count:  count,
	}).Result()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(pending))
	for _, entry := range pending {
		counts[entry.ID] = entry.RetryCount
	}

	return counts, nil
}

func (q *NotificationQueue) Ack(ctx context.Context, entry string) error {
	return q.rdb.XAck(ctx, q.stream, q.group, entry).Err()
}

func (q *NotificationQueue) Kill(
	ctx context.Context,
	entry string,
	msg notification.NotificationMessage,
	reason string,
) error {
	values, err := encodeNotificationMessage(msg)
	if err != nil {
		values = map[string]any{}
	}

	values["dead_reason"] = reason
	values["dead_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	values["dead_entry"] = entry

	if err := q.rdb.XAdd(ctx, &redis.XAddArgs{Stream: q.dead, Values: values}).Err(); err != nil {
		return err
	}

	return q.Ack(ctx, entry)
}

func (q *NotificationQueue) Pending(ctx context.Context) (int64, error) {
	pending, err := q.rdb.XPending(ctx, q.stream, q.group).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}

		return 0, err
	}

	return pending.Count, nil
}

func (q *NotificationQueue) Stats() (published, failed int64) {
	return q.published.Load(), q.failed.Load()
}

func encodeNotificationMessage(msg notification.NotificationMessage) (map[string]any, error) {
	recipients, err := json.Marshal(msg.To)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(msg.Body)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"customer_id":    msg.CustomerID,
		"message_type":   msg.MessageType,
		"template_title": msg.TemplateTitle,
		"correlation_id": msg.CorrelationID,
		"alert_id":       msg.AlertID,
		"to":             string(recipients),
		"body":           string(body),
	}, nil
}

func decodeMessages(messages []redis.XMessage) []Queued {
	queued := make([]Queued, 0, len(messages))

	for _, message := range messages {
		queued = append(queued, Queued{
			Entry:   message.ID,
			Message: decodeNotificationMessage(message.Values),
		})
	}

	return queued
}

func decodeNotificationMessage(values map[string]any) notification.NotificationMessage {
	msg := notification.NotificationMessage{
		CustomerID:    number(values, "customer_id"),
		MessageType:   field(values, "message_type"),
		TemplateTitle: field(values, "template_title"),
		CorrelationID: field(values, "correlation_id"),
		AlertID:       field(values, "alert_id"),
	}

	if recipients := field(values, "to"); recipients != "" {
		json.Unmarshal([]byte(recipients), &msg.To)
	}

	if body := field(values, "body"); body != "" {
		json.Unmarshal([]byte(body), &msg.Body)
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
