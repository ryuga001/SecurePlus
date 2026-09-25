package scanner

import (
	"context"

	"dpdp-backend/internal/datadiscovery/provider"
)

type FileJob struct {
	provider.File
	Extension string
}

type FileQueue struct {
	slots chan struct{}
	jobs  chan FileJob
}

func NewFileQueue(capacity int) *FileQueue {
	capacity = max(capacity, 1)

	return &FileQueue{
		slots: make(chan struct{}, capacity),
		jobs:  make(chan FileJob, capacity),
	}
}

func (q *FileQueue) Put(ctx context.Context, job FileJob) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case q.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	q.jobs <- job

	return nil
}

func (q *FileQueue) Jobs() <-chan FileJob {
	return q.jobs
}

func (q *FileQueue) Done() {
	<-q.slots
}

func (q *FileQueue) Close() {
	close(q.jobs)
}

func (q *FileQueue) InFlight() int {
	return len(q.slots)
}

func (q *FileQueue) Capacity() int {
	return cap(q.slots)
}
