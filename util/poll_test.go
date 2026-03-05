package util

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPollUntil_ImmediateSuccess(t *testing.T) {
	t.Parallel()
	calls := 0
	err := PollUntil(context.Background(), "test", 100*time.Millisecond, time.Second, func(ctx context.Context) (bool, error) {
		calls++
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestPollUntil_SuccessAfterRetries(t *testing.T) {
	t.Parallel()
	calls := 0
	err := PollUntil(context.Background(), "test", 10*time.Millisecond, time.Second, func(ctx context.Context) (bool, error) {
		calls++
		return calls >= 3, nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, calls)
}

func TestPollUntil_Timeout(t *testing.T) {
	t.Parallel()
	err := PollUntil(context.Background(), "slow-op", 10*time.Millisecond, 50*time.Millisecond, func(ctx context.Context) (bool, error) {
		return false, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "polling slow-op: timed out")
}

func TestPollUntil_ConditionError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	err := PollUntil(context.Background(), "failing-op", 10*time.Millisecond, time.Second, func(ctx context.Context) (bool, error) {
		return false, sentinel
	})
	require.ErrorIs(t, err, sentinel)
	require.Contains(t, err.Error(), "polling failing-op")
}

func TestPollUntil_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	err := PollUntil(ctx, "cancelled-op", 10*time.Millisecond, 5*time.Second, func(ctx context.Context) (bool, error) {
		calls++
		return false, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "polling cancelled-op")
}
