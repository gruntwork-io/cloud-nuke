package util

import (
	"context"
	"errors"
)

const (
	ExcludeFirstSeenTagKey = "exclude-first-seen-tag"
)

func GetBoolFromContext(ctx context.Context, key string) (bool, error) {
	result, ok := ctx.Value(key).(bool)
	if !ok {
		return result, errors.New("unable to find the boolean context value or correct type mismatch.")
	}

	return result, nil
}
