package util

import (
	"context"
	"errors"
)

// ContextKey is a custom type to avoid collisions when using context.WithValue
type ContextKey string

const (
	ExcludeFirstSeenTagKey ContextKey = "exclude-first-seen-tag"
	ParallelismKey         ContextKey = "parallelism"
)

// DefaultParallelism is the default number of concurrent scan/delete operations
// when --parallelism is not set (or set to 0). The work is IO-bound (AWS/GCP API
// calls), so this is a fixed value rather than being derived from CPU count
// (GOMAXPROCS), which on container runners reflects the CPU quota (e.g. 1-2 on a
// CircleCI "small"), not the concurrency this workload can usefully sustain.
const DefaultParallelism = 10

func GetBoolFromContext(ctx context.Context, key ContextKey) (bool, error) {
	result, ok := ctx.Value(key).(bool)
	if !ok {
		return result, errors.New("unable to find the boolean context value or correct type mismatch.")
	}

	return result, nil
}

// GetParallelism returns the parallelism value stored in ctx, falling back to
// DefaultParallelism if none was set (or set to a non-positive value).
func GetParallelism(ctx context.Context) int {
	if v, ok := ctx.Value(ParallelismKey).(int); ok && v > 0 {
		return v
	}
	return DefaultParallelism
}
