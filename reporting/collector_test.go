package reporting

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRenderer struct {
	mu       sync.Mutex
	events   []Event
	rendered bool
}

func (m *mockRenderer) OnEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockRenderer) Render() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rendered = true
	return nil
}

func TestCollector_FullFlow(t *testing.T) {
	c := NewCollector()
	r := &mockRenderer{}
	c.AddRenderer(r)
	c.AddRenderer(nil) // nil should be ignored

	// Record events
	c.RecordFound("ec2", "us-east-1", "i-1", true, "")
	c.RecordFound("ec2", "us-east-1", "i-2", false, "protected")
	c.RecordDeleted("ec2", "us-east-1", "i-1", nil)
	c.RecordDeleted("ec2", "us-east-1", "i-3", errors.New("access denied"))
	c.RecordError("s3", "Unable to list", errors.New("timeout"))

	require.Len(t, r.events, 5)

	// Verify event types
	assert.IsType(t, ResourceFound{}, r.events[0])
	assert.IsType(t, ResourceDeleted{}, r.events[2])
	assert.IsType(t, GeneralError{}, r.events[4])

	// Complete should render
	err := c.Complete()
	assert.NoError(t, err)
	assert.True(t, r.rendered)

	// Events after close should be ignored
	c.RecordDeleted("ec2", "us-east-1", "i-4", nil)
	assert.Len(t, r.events, 5)

	// Second complete should be no-op
	assert.NoError(t, c.Complete())
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	c := NewCollector()
	r := &mockRenderer{}
	c.AddRenderer(r)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c.RecordDeleted("ec2", "us-east-1", "i-test", nil)
		}(i)
	}
	wg.Wait()

	assert.Len(t, r.events, 100)
}
