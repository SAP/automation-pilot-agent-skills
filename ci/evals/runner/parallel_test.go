package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelJobsBoundReservationsAndPersistEveryOutcome(t *testing.T) {
	var active, peak atomic.Int32
	results := make([]int, 7)
	calls := 0
	err := parallelJobs(context.Background(), 7, 8, 3, 1, func(index int) int {
		n := active.Add(1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return 1
	}, func(index int, reason string) int { return -1 }, func(value int) float64 {
		if value == 1 {
			return 1
		}
		return 0
	}, func(index, value int) error { results[index] = value; calls++; return nil })
	if err != nil || calls != 7 || peak.Load() > 3 {
		t.Fatal(err, calls, peak.Load())
	}
	passed, blocked := 0, 0
	for _, r := range results {
		if r == 1 {
			passed++
		}
		if r == -1 {
			blocked++
		}
	}
	if passed != 3 || blocked != 4 {
		t.Fatal(results)
	}
}

func TestParallelJobsCancellationProducesUnrunRecords(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := parallelJobs(ctx, 4, 2, 0, 1, func(index int) int { t.Error("work started after cancellation"); return 1 }, func(index int, reason string) int { return -1 }, func(value int) float64 { return 0 }, func(index, value int) error {
		if value != -1 {
			t.Error(value)
		}
		calls++
		return nil
	})
	if err != nil || calls != 4 {
		t.Fatal(err, calls)
	}
}
