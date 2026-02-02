package reporting

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.Len(t, r.events, 5)

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

