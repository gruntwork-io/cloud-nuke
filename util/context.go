package util

import (
	"context"
	"errors"
)

// ContextKey is a custom type to avoid collisions when using context.WithValue
type ContextKey string

const (
	ExcludeFirstSeenTagKey ContextKey = "exclude-first-seen-tag"
)

func GetBoolFromContext(ctx context.Context, key ContextKey) (bool, error) {
	result, ok := ctx.Value(key).(bool)
	if !ok {
		return result, errors.New("unable to find the boolean context value or correct type mismatch.")
	}

	return result, nil
}
