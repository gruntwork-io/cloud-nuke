package commands

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTimeoutAppliesTheFlagAsADeadline(t *testing.T) {
	t.Parallel()

	timeout := 30 * time.Minute
	ctx, cancel := withTimeout(context.Background(), &timeout)
	defer cancel()

	deadline, ok := ctx.Deadline()
	require.True(t, ok, "--timeout should put a deadline on the run")
	assert.WithinDuration(t, time.Now().Add(timeout), deadline, time.Minute)
}

func TestWithTimeoutLeavesTheContextAloneWithoutTheFlag(t *testing.T) {
	t.Parallel()

	zero := time.Duration(0)
	for _, timeout := range []*time.Duration{nil, &zero} {
		ctx, cancel := withTimeout(context.Background(), timeout)
		_, ok := ctx.Deadline()
		assert.False(t, ok, "no --timeout means no deadline")
		cancel()
	}
}

func TestTimeoutErrorNamesTheDeadlineItPassed(t *testing.T) {
	t.Parallel()

	timeout := 10 * time.Minute

	assert.NoError(t, timeoutError(nil, &timeout))

	other := errors.New("subscription not found")
	assert.Equal(t, other, timeoutError(other, &timeout))

	passed := timeoutError(context.DeadlineExceeded, &timeout)
	require.Error(t, passed)
	assert.Contains(t, passed.Error(), "execution timed out after: 10m0s")
	assert.ErrorIs(t, passed, context.DeadlineExceeded, "the deadline is named beside the cause, not in place of it")
}

func TestTimeoutErrorKeepsWhatElseTheRunReported(t *testing.T) {
	t.Parallel()

	// A run that failed three deletes and then ran out of time has to report
	// both: replacing the error with the deadline loses every failure.
	timeout := 10 * time.Minute
	failed := fmt.Errorf("deleting rg-1: %w", errors.New("still has a lock"))
	both := errors.Join(failed, context.DeadlineExceeded)

	named := timeoutError(both, &timeout)
	require.Error(t, named)
	assert.Contains(t, named.Error(), "execution timed out after: 10m0s")
	assert.Contains(t, named.Error(), "still has a lock")
}

func TestTimedOutReadsADeadlineTheScanPassedWithoutFailing(t *testing.T) {
	t.Parallel()

	// The engine records a listing failure and carries on, which is right for
	// one resource type among many and wrong for a deadline: without this, a
	// run whose lists were cut short reports a clean subscription.
	timeout := time.Minute

	passed, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := timedOut(passed, &timeout)
	require.Error(t, err)
	assert.Equal(t, util.ResourceExecutionTimeout{Timeout: timeout}, err)

	// A run still inside its deadline, and a cancelled one, are not timeouts.
	assert.NoError(t, timedOut(context.Background(), &timeout))
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	assert.NoError(t, timedOut(cancelled, &timeout))
	assert.NoError(t, timedOut(passed, nil))
}
