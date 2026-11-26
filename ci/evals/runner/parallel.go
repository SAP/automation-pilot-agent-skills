package main

import (
	"context"
	"fmt"
)

// The controller alone charges budget reservations and writes checkpoints.
// Workers only operate on independent case or grader workspaces.
func parallelJobs[T any](ctx context.Context, count, workers int, budget, reservation float64, work func(int) T, blocked func(int, string) T, charge func(T) float64, consume func(int, T) error) error {
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		return fmt.Errorf("parallelism must be between 1 and 8")
	}
	type completion struct {
		index int
		value T
	}
	done := make(chan completion, workers)
	next, active := 0, 0
	spent := 0.0
	var firstErr error
	for next < count || active > 0 {
		for firstErr == nil && ctx.Err() == nil && next < count && active < workers && (budget == 0 || spent+float64(active+1)*reservation <= budget) {
			index := next
			next++
			active++
			go func() { done <- completion{index, work(index)} }()
		}
		if active == 0 {
			for next < count {
				reason := "run cost ceiling reached before next session"
				if ctx.Err() != nil {
					reason = "run cancelled"
				}
				if firstErr != nil {
					reason = "checkpoint failed"
				}
				if err := consume(next, blocked(next, reason)); err != nil && firstErr == nil {
					firstErr = err
				}
				next++
			}
			break
		}
		result := <-done
		active--
		spent += charge(result.value)
		if err := consume(result.index, result.value); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
