package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"dpdp-backend/internal/config"
)

const statsInterval = time.Minute

type WorkerPool struct {
	queue     *NotificationQueue
	processor *NotificationProcessor
	cfg       config.Notification

	wg sync.WaitGroup

	processed atomic.Int64
	failures  atomic.Int64
}

func NewWorkerPool(queue *NotificationQueue, processor *NotificationProcessor, cfg config.Notification) *WorkerPool {
	return &WorkerPool{queue: queue, processor: processor, cfg: cfg}
}

func (p *WorkerPool) Start(ctx context.Context) {
	for index := range p.cfg.Workers {
		consumer := fmt.Sprintf("notification-worker-%d", index+1)

		p.wg.Add(1)

		go func() {
			defer p.wg.Done()

			p.consume(ctx, consumer)
		}()
	}

	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		p.reclaim(ctx)
	}()

	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		p.report(ctx)
	}()

	slog.InfoContext(ctx, "notification workers started",
		"workers", p.cfg.Workers,
		"stream", p.cfg.Stream,
		"group", p.cfg.ConsumerGroup,
	)
}

func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

func (p *WorkerPool) Stats() (processed, failures int64) {
	return p.processed.Load(), p.failures.Load()
}

func (p *WorkerPool) consume(ctx context.Context, consumer string) {
	for {
		if ctx.Err() != nil {
			return
		}

		queued, err := p.queue.Consume(ctx, consumer, p.cfg.BatchSize, p.cfg.BlockTime)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			slog.ErrorContext(ctx, "notification queue read failed", "worker", consumer, "error", err)
			p.pause(ctx)

			continue
		}

		for _, entry := range queued {
			p.handle(ctx, consumer, entry)
		}
	}
}

func (p *WorkerPool) reclaim(ctx context.Context) {
	const consumer = "notification-reclaimer"

	ticker := time.NewTicker(p.cfg.ClaimIdle)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		queued, err := p.queue.Reclaim(ctx, consumer, p.cfg.ClaimIdle, p.cfg.BatchSize)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			slog.ErrorContext(ctx, "notification queue reclaim failed", "error", err)

			continue
		}

		for _, entry := range queued {
			slog.WarnContext(ctx, "reclaimed stalled notification",
				"entry", entry.Entry,
				"correlation_id", entry.Message.CorrelationID,
				"retries", entry.Retries,
			)

			p.handle(ctx, consumer, entry)
		}
	}
}

func (p *WorkerPool) handle(ctx context.Context, consumer string, entry Queued) {
	msg := entry.Message

	if msg.MessageType == "" {
		slog.ErrorContext(ctx, "discarding unreadable notification entry",
			"entry", entry.Entry,
			"worker", consumer,
		)
		p.kill(ctx, consumer, entry, "unreadable entry")

		return
	}

	if entry.Retries > p.cfg.MaxDeliveries {
		slog.ErrorContext(ctx, "notification exceeded max deliveries",
			"entry", entry.Entry,
			"worker", consumer,
			"correlation_id", msg.CorrelationID,
			"alert_id", msg.AlertID,
			"retries", entry.Retries,
		)
		p.kill(ctx, consumer, entry, "max deliveries exceeded")

		return
	}

	started := time.Now()
	err := p.processor.Process(context.WithoutCancel(ctx), msg)
	duration := time.Since(started)

	if errors.Is(err, ErrUnprocessable) {
		p.failures.Add(1)

		slog.ErrorContext(ctx, "notification is unprocessable, moving to dead stream",
			"entry", entry.Entry,
			"worker", consumer,
			"correlation_id", msg.CorrelationID,
			"alert_id", msg.AlertID,
			"error", err,
		)
		p.kill(ctx, consumer, entry, err.Error())

		return
	}

	if err != nil {
		p.failures.Add(1)

		slog.ErrorContext(ctx, "notification not sent, leaving entry pending",
			"entry", entry.Entry,
			"worker", consumer,
			"correlation_id", msg.CorrelationID,
			"alert_id", msg.AlertID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)

		return
	}

	p.processed.Add(1)

	slog.InfoContext(ctx, "notification sent",
		"entry", entry.Entry,
		"worker", consumer,
		"customer_id", msg.CustomerID,
		"correlation_id", msg.CorrelationID,
		"alert_id", msg.AlertID,
		"template", msg.TemplateTitle,
		"duration_ms", duration.Milliseconds(),
	)

	p.ack(ctx, consumer, entry)
}

func (p *WorkerPool) ack(ctx context.Context, consumer string, entry Queued) {
	if err := p.queue.Ack(context.WithoutCancel(ctx), entry.Entry); err != nil {
		slog.ErrorContext(ctx, "notification ack failed",
			"entry", entry.Entry,
			"worker", consumer,
			"correlation_id", entry.Message.CorrelationID,
			"error", err,
		)
	}
}

func (p *WorkerPool) kill(ctx context.Context, consumer string, entry Queued, reason string) {
	err := p.queue.Kill(context.WithoutCancel(ctx), entry.Entry, entry.Message, reason)
	if err != nil {
		slog.ErrorContext(ctx, "notification dead stream write failed",
			"entry", entry.Entry,
			"worker", consumer,
			"reason", reason,
			"error", err,
		)
	}
}

func (p *WorkerPool) report(ctx context.Context) {
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		published, publishErrors := p.queue.Stats()
		processed, failures := p.Stats()

		pending, err := p.queue.Pending(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			slog.WarnContext(ctx, "notification queue pending lookup failed", "error", err)

			continue
		}

		slog.InfoContext(ctx, "notification queue stats",
			"published_total", published,
			"publish_errors_total", publishErrors,
			"processing_total", processed,
			"processing_errors_total", failures,
			"pending", pending,
		)
	}
}

func (p *WorkerPool) pause(ctx context.Context) {
	timer := time.NewTimer(p.cfg.BlockTime)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
