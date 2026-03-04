package util

import (
	"context"
	"fmt"
	"time"
)

// PollUntil repeatedly calls condition until it returns true, the timeout
// elapses, or the context is cancelled. The condition is checked immediately
// on the first iteration (before any sleep). This is useful for waiting on
// async AWS operations that lack a built-in SDK waiter (e.g., VPN gateway
// detachment).
//
// The interval is the delay between condition checks. The caller must ensure
// interval > 0 to avoid a hot spin loop.
func PollUntil(ctx context.Context, description string, interval, timeout time.Duration, condition func(ctx context.Context) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		done, err := condition(ctx)
		if err != nil {
			return fmt.Errorf("polling %s: %w", description, err)
		}
		if done {
			return nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("polling %s: timed out after %s", description, timeout)
			}
			return fmt.Errorf("polling %s: %w", description, ctx.Err())
		case <-timer.C:
		}
	}
}
