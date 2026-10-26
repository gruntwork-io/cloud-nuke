package commands

import (
	"context"
	"errors"
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

	timedOut := timeoutError(context.DeadlineExceeded, &timeout)
	assert.Equal(t, util.ResourceExecutionTimeout{Timeout: timeout}, timedOut)
	assert.Equal(t, "execution timed out after: 10m0s", timedOut.Error())
}
