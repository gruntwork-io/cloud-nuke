package reporting

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRenderer struct {
	mu     sync.Mutex
	events []Event
}

func (m *mockRenderer) OnEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func TestCollector_FullFlow(t *testing.T) {
	c := NewCollector()
	r := &mockRenderer{}
	c.AddRenderer(r)
	c.AddRenderer(nil) // nil should be ignored

	// Emit events using the single Emit method
	c.Emit(ResourceFound{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-1",
		Nukable:      true,
	})
	c.Emit(ResourceFound{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-2",
		Nukable:      false,
		Reason:       "protected",
	})
	c.Emit(ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-1",
		Success:      true,
	})
	c.Emit(ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-3",
		Success:      false,
		Error:        "access denied",
	})
	c.Emit(GeneralError{
		ResourceType: "s3",
		Description:  "Unable to list",
		Error:        "timeout",
	})

	require.Len(t, r.events, 5)

	// Verify event types
	assert.IsType(t, ResourceFound{}, r.events[0])
	assert.IsType(t, ResourceDeleted{}, r.events[2])
	assert.IsType(t, GeneralError{}, r.events[4])

	// Complete marks collector as closed
	c.Complete()

	// Events after close should be ignored
	c.Emit(ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-4",
		Success:      true,
	})
	assert.Len(t, r.events, 5)

	// Second complete should be no-op
	c.Complete()
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	c := NewCollector()
	r := &mockRenderer{}
	c.AddRenderer(r)

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Emit(ResourceDeleted{
				ResourceType: "ec2",
				Region:       "us-east-1",
				Identifier:   "i-test",
				Success:      true,
			})
		}()
	}
	wg.Wait()

	assert.Len(t, r.events, 100)
}

func TestCollector_ProgressEvents(t *testing.T) {
	c := NewCollector()
	r := &mockRenderer{}
	c.AddRenderer(r)

	// Test progress events
	c.Emit(NukeStarted{Total: 10})
	c.Emit(NukeProgress{
		ResourceType: "ec2",
		Region:       "us-east-1",
		BatchSize:    5,
	})
	c.Emit(ScanProgress{
		ResourceType: "s3",
		Region:       "us-west-2",
	})

	require.Len(t, r.events, 3)

	// Verify event types
	assert.IsType(t, NukeStarted{}, r.events[0])
	assert.IsType(t, NukeProgress{}, r.events[1])
	assert.IsType(t, ScanProgress{}, r.events[2])

	// Verify NukeStarted content
	nukeStarted := r.events[0].(NukeStarted)
	assert.Equal(t, 10, nukeStarted.Total)

	// Verify NukeProgress content
	nukeProgress := r.events[1].(NukeProgress)
	assert.Equal(t, "ec2", nukeProgress.ResourceType)
	assert.Equal(t, 5, nukeProgress.BatchSize)
}
