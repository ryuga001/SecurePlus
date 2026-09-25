package scanner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/scanner"
)

func job(key string) scanner.FileJob {
	return scanner.FileJob{File: provider.File{Key: key, Name: key}, Extension: "txt"}
}

func TestFileQueuePutWithinCapacity(t *testing.T) {
	queue := scanner.NewFileQueue(2)

	for _, key := range []string{"F11", "F12"} {
		if err := queue.Put(context.Background(), job(key)); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	if queue.InFlight() != 2 {
		t.Fatalf("in flight = %d, want 2", queue.InFlight())
	}
}

func TestFileQueueHoldsSlotUntilDone(t *testing.T) {
	queue := scanner.NewFileQueue(2)
	ctx := context.Background()

	_ = queue.Put(ctx, job("F11"))
	_ = queue.Put(ctx, job("F12"))

	taken := <-queue.Jobs()
	if taken.Key != "F11" {
		t.Fatalf("first job = %s", taken.Key)
	}

	admitted := make(chan struct{})

	go func() {
		_ = queue.Put(ctx, job("F13"))
		close(admitted)
	}()

	select {
	case <-admitted:
		t.Fatal("F13 was admitted while F11 was still in flight")
	case <-time.After(50 * time.Millisecond):
	}

	queue.Done()

	select {
	case <-admitted:
	case <-time.After(time.Second):
		t.Fatal("F13 was not admitted after F11 completed")
	}

	if next := <-queue.Jobs(); next.Key != "F12" {
		t.Fatalf("second job = %s", next.Key)
	}

	if next := <-queue.Jobs(); next.Key != "F13" {
		t.Fatalf("third job = %s", next.Key)
	}
}

func TestFileQueuePutHonoursCancellation(t *testing.T) {
	queue := scanner.NewFileQueue(1)
	_ = queue.Put(context.Background(), job("F11"))

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	go func() { result <- queue.Put(ctx, job("F12")) }()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("put error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked put did not return after cancellation")
	}

	if queue.InFlight() != 1 {
		t.Fatalf("in flight = %d, cancelled put must not hold a slot", queue.InFlight())
	}
}

func TestFileQueueCloseEndsConsumer(t *testing.T) {
	queue := scanner.NewFileQueue(3)
	ctx := context.Background()

	_ = queue.Put(ctx, job("F11"))
	_ = queue.Put(ctx, job("F12"))
	queue.Close()

	seen := 0
	for range queue.Jobs() {
		seen++
		queue.Done()
	}

	if seen != 2 || queue.InFlight() != 0 {
		t.Fatalf("seen = %d in flight = %d", seen, queue.InFlight())
	}
}
