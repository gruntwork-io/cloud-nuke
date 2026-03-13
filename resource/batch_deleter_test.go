package resource

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ctx    = context.Background()
	client = &mockClient{}
	scope  = Scope{}
)

func ids(ss ...string) []*string { return util.ToStringPtrSlice(ss) }

func noopDelete(_ context.Context, _ *mockClient, _ *string) error { return nil }
func failDelete(_ context.Context, _ *mockClient, _ *string) error { return errors.New("fail") }

func selectiveDelete(_ context.Context, _ *mockClient, id *string) error {
	if *id == "fail" {
		return errors.New("delete failed")
	}
	return nil
}

// BulkDeleter

func TestBulkDeleter_Success(t *testing.T) {
	results := BulkDeleter(func(_ context.Context, _ *mockClient, _ []string) error {
		return nil
	})(ctx, client, scope, "test", ids("a", "b", "c"))

	require.Len(t, results, 3)
	for _, r := range results {
		assert.NoError(t, r.Error)
	}
}

func TestBulkDeleter_Error(t *testing.T) {
	results := BulkDeleter(func(_ context.Context, _ *mockClient, _ []string) error {
		return errors.New("bulk fail")
	})(ctx, client, scope, "test", ids("a", "b"))

	require.Len(t, results, 2)
	for _, r := range results {
		assert.ErrorContains(t, r.Error, "bulk fail")
	}
}

// BulkResultDeleter

func TestBulkResultDeleter_PartialFailure(t *testing.T) {
	results := BulkResultDeleter(func(_ context.Context, _ *mockClient, ids []string) []NukeResult {
		out := make([]NukeResult, len(ids))
		for i, id := range ids {
			if id == "bad" {
				out[i] = NukeResult{Identifier: id, Error: errors.New("failed")}
			} else {
				out[i] = NukeResult{Identifier: id}
			}
		}
		return out
	})(ctx, client, scope, "test", ids("ok", "bad", "ok2"))

	require.Len(t, results, 3)
	assert.NoError(t, results[0].Error)
	assert.Error(t, results[1].Error)
	assert.NoError(t, results[2].Error)
}

// DeleteThenWait

func TestDeleteThenWait(t *testing.T) {
	t.Run("success calls wait", func(t *testing.T) {
		waitCalled := false
		fn := DeleteThenWait(noopDelete, func(_ context.Context, _ *mockClient, _ *string) error {
			waitCalled = true
			return nil
		})
		id := "x"
		assert.NoError(t, fn(ctx, client, &id))
		assert.True(t, waitCalled)
	})
	t.Run("delete error skips wait", func(t *testing.T) {
		waitCalled := false
		fn := DeleteThenWait(failDelete, func(_ context.Context, _ *mockClient, _ *string) error {
			waitCalled = true
			return nil
		})
		id := "x"
		assert.Error(t, fn(ctx, client, &id))
		assert.False(t, waitCalled)
	})
	t.Run("wait error propagates", func(t *testing.T) {
		fn := DeleteThenWait(noopDelete, func(_ context.Context, _ *mockClient, _ *string) error {
			return errors.New("wait timeout")
		})
		id := "x"
		assert.ErrorContains(t, fn(ctx, client, &id), "wait timeout")
	})
}

// SequentialDeleteThenWaitAll

func TestSequentialDeleteThenWaitAll(t *testing.T) {
	t.Run("success preserves order", func(t *testing.T) {
		var order []string
		results := SequentialDeleteThenWaitAll(
			func(_ context.Context, _ *mockClient, id *string) error { order = append(order, *id); return nil },
			func(_ context.Context, _ *mockClient, ids []string) error { assert.Equal(t, order, ids); return nil },
		)(ctx, client, scope, "test", ids("a", "b", "c"))

		require.Len(t, results, 3)
		assert.Equal(t, []string{"a", "b", "c"}, order)
	})
	t.Run("mixed failures only waits successes", func(t *testing.T) {
		results := SequentialDeleteThenWaitAll(
			selectiveDelete,
			func(_ context.Context, _ *mockClient, waited []string) error {
				assert.Equal(t, []string{"ok1", "ok2"}, waited)
				return nil
			},
		)(ctx, client, scope, "test", ids("ok1", "fail", "ok2"))

		require.Len(t, results, 3)
		errCount := 0
		for _, r := range results {
			if r.Error != nil {
				errCount++
			}
		}
		assert.Equal(t, 1, errCount)
	})
	t.Run("wait error applied to all successes", func(t *testing.T) {
		results := SequentialDeleteThenWaitAll(
			noopDelete,
			func(_ context.Context, _ *mockClient, _ []string) error { return errors.New("wait failed") },
		)(ctx, client, scope, "test", ids("a", "b"))

		require.Len(t, results, 2)
		for _, r := range results {
			assert.ErrorContains(t, r.Error, "wait failed")
		}
	})
}

// ConcurrentDeleteThenWaitAll

func TestConcurrentDeleteThenWaitAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var count atomic.Int32
		results := ConcurrentDeleteThenWaitAll(
			func(_ context.Context, _ *mockClient, _ *string) error { count.Add(1); return nil },
			func(_ context.Context, _ *mockClient, waited []string) error { assert.Len(t, waited, 3); return nil },
		)(ctx, client, scope, "test", ids("a", "b", "c"))

		require.Len(t, results, 3)
		assert.Equal(t, int32(3), count.Load())
	})
	t.Run("mixed failures only waits successes", func(t *testing.T) {
		results := ConcurrentDeleteThenWaitAll(
			selectiveDelete,
			func(_ context.Context, _ *mockClient, waited []string) error { assert.Len(t, waited, 2); return nil },
		)(ctx, client, scope, "test", ids("ok1", "fail", "ok2"))

		require.Len(t, results, 3)
		errCount := 0
		for _, r := range results {
			if r.Error != nil {
				errCount++
			}
		}
		assert.Equal(t, 1, errCount)
	})
	t.Run("all fail skips wait", func(t *testing.T) {
		waitCalled := false
		results := ConcurrentDeleteThenWaitAll(
			failDelete,
			func(_ context.Context, _ *mockClient, _ []string) error { waitCalled = true; return nil },
		)(ctx, client, scope, "test", ids("a", "b"))

		require.Len(t, results, 2)
		assert.False(t, waitCalled)
	})
}

// SimpleBatchDeleter extras (main tests in resource_test.go)

func TestSimpleBatchDeleter_ErrorPropagation(t *testing.T) {
	results := SimpleBatchDeleter(selectiveDelete)(ctx, client, scope, "test", ids("good", "fail", "good2"))
	require.Len(t, results, 3)
	assert.NoError(t, results[0].Error)
	assert.Error(t, results[1].Error)
	assert.NoError(t, results[2].Error)
}

func TestSimpleBatchDeleter_Concurrency(t *testing.T) {
	var count atomic.Int32
	deleter := SimpleBatchDeleter(func(_ context.Context, _ *mockClient, _ *string) error {
		count.Add(1)
		return nil
	})
	// 25 items exceeds DefaultMaxConcurrent to exercise semaphore
	items := make([]*string, 25)
	for i := range items {
		s := "id-" + string(rune('a'+i))
		items[i] = &s
	}
	results := deleter(ctx, client, scope, "test", items)
	assert.Len(t, results, 25)
	assert.Equal(t, int32(25), count.Load())
}

// Empty input tests — all NukerFunc types should return nil

func TestNukerFuncs_Empty(t *testing.T) {
	nukers := map[string]NukerFunc[*mockClient]{
		"SimpleBatch":              SimpleBatchDeleter(noopDelete),
		"Sequential":              SequentialDeleter(noopDelete),
		"MultiStep":               MultiStepDeleter(noopDelete),
		"Bulk":                    BulkDeleter(func(_ context.Context, _ *mockClient, _ []string) error { return nil }),
		"BulkResult":              BulkResultDeleter(func(_ context.Context, _ *mockClient, _ []string) []NukeResult { return nil }),
		"SequentialDeleteWaitAll": SequentialDeleteThenWaitAll(noopDelete, func(_ context.Context, _ *mockClient, _ []string) error { return nil }),
		"ConcurrentDeleteWaitAll": ConcurrentDeleteThenWaitAll(noopDelete, func(_ context.Context, _ *mockClient, _ []string) error { return nil }),
	}
	for name, nuker := range nukers {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, nuker(ctx, client, scope, "test", nil))
		})
	}
}

// logStart

func TestLogStart(t *testing.T) {
	assert.True(t, logStart(nil, "test", scope))
	id := "x"
	assert.False(t, logStart([]*string{&id}, "test", scope))
}
