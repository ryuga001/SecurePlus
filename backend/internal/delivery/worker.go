package delivery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"dpdp-backend/internal/config"
)

const statsInterval = time.Minute

type WorkerPool struct {
	queue   *Queue
	service *DeliveryService
	cfg     config.Delivery

	wg sync.WaitGroup

	processed atomic.Int64
	failures  atomic.Int64
}

func NewWorkerPool(queue *Queue, service *DeliveryService, cfg config.Delivery) *WorkerPool {
	return &WorkerPool{queue: queue, service: service, cfg: cfg}
}

func (p *WorkerPool) Start(ctx context.Context) {
	for index := range p.cfg.Workers {
		consumer := fmt.Sprintf("delivery-worker-%d", index+1)

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

	slog.InfoContext(ctx, "delivery workers started",
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

			slog.ErrorContext(ctx, "delivery queue read failed", "worker", consumer, "error", err)
			p.pause(ctx)

			continue
		}

		for _, entry := range queued {
			p.handle(ctx, consumer, entry)
		}
	}
}

func (p *WorkerPool) reclaim(ctx context.Context) {
	const consumer = "delivery-reclaimer"

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

			slog.ErrorContext(ctx, "delivery queue reclaim failed", "error", err)

			continue
		}

		for _, entry := range queued {
			slog.WarnContext(ctx, "reclaimed stalled delivery",
				"entry", entry.Entry,
				"correlation_id", entry.Message.CorrelationID,
			)

			p.handle(ctx, consumer, entry)
		}
	}
}

func (p *WorkerPool) handle(ctx context.Context, consumer string, entry Queued) {
	msg := entry.Message
	started := time.Now()

	if msg.CorrelationID == "" {
		slog.ErrorContext(ctx, "discarding unreadable queue entry", "entry", entry.Entry, "worker", consumer)
		p.ack(ctx, consumer, entry)

		return
	}

	err := p.service.Process(context.WithoutCancel(ctx), msg)

	duration := time.Since(started)

	if err != nil {
		p.failures.Add(1)

		slog.ErrorContext(ctx, "delivery not recorded, leaving entry pending",
			"entry", entry.Entry,
			"worker", consumer,
			"message_id", msg.MessageID,
			"correlation_id", msg.CorrelationID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)

		return
	}

	p.processed.Add(1)

	slog.InfoContext(ctx, "delivery processed",
		"entry", entry.Entry,
		"worker", consumer,
		"message_id", msg.MessageID,
		"correlation_id", msg.CorrelationID,
		"duration_ms", duration.Milliseconds(),
	)

	p.ack(ctx, consumer, entry)
}

func (p *WorkerPool) ack(ctx context.Context, consumer string, entry Queued) {
	if err := p.queue.Ack(context.WithoutCancel(ctx), entry.Entry); err != nil {
		slog.ErrorContext(ctx, "delivery ack failed",
			"entry", entry.Entry,
			"worker", consumer,
			"correlation_id", entry.Message.CorrelationID,
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

			slog.WarnContext(ctx, "delivery queue pending lookup failed", "error", err)

			continue
		}

		slog.InfoContext(ctx, "delivery queue stats",
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
