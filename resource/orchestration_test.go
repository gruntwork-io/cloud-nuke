package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNukeableResource implements NukeableResource for testing.
// Set getErr to make GetAndSetIdentifiers fail.
// Set nukableIDs (non-nil) to control per-ID IsNukable behavior.
type mockNukeableResource struct {
	name        string
	identifiers []string
	batchSize   int
	nukeErr     error
	nukeCalled  int
	getErr      error
	nukableIDs  map[string]bool // nil = all nukable
}

func (m *mockNukeableResource) ResourceName() string          { return m.name }
func (m *mockNukeableResource) ResourceIdentifiers() []string { return m.identifiers }
func (m *mockNukeableResource) MaxBatchSize() int {
	if m.batchSize > 0 {
		return m.batchSize
	}
	return DefaultBatchSize
}
func (m *mockNukeableResource) Nuke(ctx context.Context, ids []string) ([]NukeResult, error) {
	m.nukeCalled++
	results := make([]NukeResult, len(ids))
	for i, id := range ids {
		results[i] = NukeResult{Identifier: id, Error: m.nukeErr}
	}
	if m.nukeErr != nil {
		return results, m.nukeErr
	}
	return results, nil
}
func (m *mockNukeableResource) GetAndSetIdentifiers(ctx context.Context, cfg config.Config) ([]string, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.identifiers, nil
}
func (m *mockNukeableResource) IsNukable(id string) (bool, error) {
	if m.nukableIDs != nil {
		if m.nukableIDs[id] {
			return true, nil
		}
		return false, errors.New("not nukable")
	}
	return true, nil
}
func (m *mockNukeableResource) GetAndSetResourceConfig(cfg config.Config) config.ResourceType {
	return config.ResourceType{}
}

// --- ScanResource tests ---

func TestScanResource(t *testing.T) {
	res := &mockNukeableResource{name: "test", identifiers: []string{"id-1", "id-2"}}

	var found []string
	cb := ScanCallbacks{
		OnResourceFound: func(_, _, id string, _ bool, _ string) { found = append(found, id) },
	}

	ids := ScanResource(context.Background(), res, "us-east-1", config.Config{}, cb)

	assert.Equal(t, []string{"id-1", "id-2"}, ids)
	assert.Equal(t, []string{"id-1", "id-2"}, found)
}

func TestScanResource_ErrorCallsOnScanError(t *testing.T) {
	res := &mockNukeableResource{name: "failing", getErr: errors.New("access denied")}

	var capturedErr error
	cb := ScanCallbacks{
		OnScanError: func(_, _ string, err error) { capturedErr = err },
	}

	ids := ScanResource(context.Background(), res, "us-east-1", config.Config{}, cb)

	assert.Nil(t, ids)
	require.Error(t, capturedErr)
	assert.Contains(t, capturedErr.Error(), "access denied")
}

func TestScanResource_ErrorSkipped(t *testing.T) {
	res := &mockNukeableResource{name: "disabled", getErr: errors.New("API disabled")}

	scanErrorCalled := false
	cb := ScanCallbacks{
		OnScanError:     func(_, _ string, _ error) { scanErrorCalled = true },
		ShouldSkipError: func(_ string, _ error) bool { return true },
	}

	ids := ScanResource(context.Background(), res, "us-east-1", config.Config{}, cb)

	assert.Nil(t, ids)
	assert.False(t, scanErrorCalled)
}

// --- NukeInBatches tests ---

func TestNukeInBatches(t *testing.T) {
	res := &mockNukeableResource{name: "test", identifiers: []string{"a", "b", "c"}, batchSize: 2}

	var resultCount int
	cb := NukeBatchCallbacks{
		OnResult: func(_, _ string, _ NukeResult) { resultCount++ },
	}

	require.NoError(t, NukeInBatches(context.Background(), res, "us-east-1", cb))
	assert.Equal(t, 3, resultCount)
}

func TestNukeInBatches_NothingToNuke(t *testing.T) {
	tests := map[string]*mockNukeableResource{
		"empty":        {name: "empty", identifiers: []string{}},
		"all_filtered": {name: "denied", identifiers: []string{"d1", "d2"}, nukableIDs: map[string]bool{}},
	}
	for name, res := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, NukeInBatches(context.Background(), res, "us-east-1", NukeBatchCallbacks{}))
			assert.Equal(t, 0, res.nukeCalled)
		})
	}
}

func TestNukeInBatches_Error(t *testing.T) {
	res := &mockNukeableResource{name: "failing", identifiers: []string{"id-1"}, nukeErr: errors.New("delete failed")}

	err := NukeInBatches(context.Background(), res, "us-east-1", NukeBatchCallbacks{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

func TestNukeInBatches_FiltersNonNukable(t *testing.T) {
	res := &mockNukeableResource{
		name:        "mixed",
		identifiers: []string{"allowed-1", "denied-1", "allowed-2", "denied-2"},
		nukableIDs:  map[string]bool{"allowed-1": true, "allowed-2": true},
	}

	var nukedIDs []string
	cb := NukeBatchCallbacks{
		OnResult: func(_, _ string, r NukeResult) { nukedIDs = append(nukedIDs, r.Identifier) },
	}

	require.NoError(t, NukeInBatches(context.Background(), res, "us-east-1", cb))
	assert.Equal(t, []string{"allowed-1", "allowed-2"}, nukedIDs)
}
