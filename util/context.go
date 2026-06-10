package util

import (
	"context"
	"errors"
	"runtime"
)

// ContextKey is a custom type to avoid collisions when using context.WithValue
type ContextKey string

const (
	ExcludeFirstSeenTagKey ContextKey = "exclude-first-seen-tag"
	ParallelismKey         ContextKey = "parallelism"
)

func GetBoolFromContext(ctx context.Context, key ContextKey) (bool, error) {
	result, ok := ctx.Value(key).(bool)
	if !ok {
		return result, errors.New("unable to find the boolean context value or correct type mismatch.")
	}

	return result, nil
}

// GetParallelism returns the parallelism value stored in ctx, falling back to
// runtime.GOMAXPROCS(0) if none was set.
func GetParallelism(ctx context.Context) int {
	if v, ok := ctx.Value(ParallelismKey).(int); ok && v > 0 {
		return v
	}
	return runtime.GOMAXPROCS(0)
}
