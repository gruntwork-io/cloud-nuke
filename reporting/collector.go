package reporting

import (
	"sync"
)

// Renderer processes events and produces output.
// CLI renderer outputs progressively on ScanComplete and NukeComplete.
// JSON renderer outputs once on Complete (emitted by collector.Complete()).
type Renderer interface {
	// OnEvent is called for each event as it occurs.
	OnEvent(event Event)
}

// Collector receives events and routes them to renderers.
// Thread-safe for concurrent event emission.
type Collector struct {
	mu        sync.Mutex
	renderers []Renderer
	closed    bool
}

// NewCollector creates a new Collector.
func NewCollector() *Collector {
	return &Collector{
		renderers: make([]Renderer, 0),
	}
}

// AddRenderer adds a renderer to receive events.
// Must be called during setup before any concurrent operations.
func (c *Collector) AddRenderer(r Renderer) {
	if r == nil {
		return
	}
	c.renderers = append(c.renderers, r)
}

// Emit sends an event to all renderers.
// Callers construct event structs directly and call this method.
func (c *Collector) Emit(event Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	for _, r := range c.renderers {
		r.OnEvent(event)
	}
}

// Complete marks collection as finished and signals renderers to flush output.
// Emits Complete event before closing, allowing renderers to output final state.
// Safe to call multiple times - subsequent calls are no-ops.
func (c *Collector) Complete() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	// Signal renderers to output final state
	for _, r := range c.renderers {
		r.OnEvent(Complete{})
	}

	c.closed = true
}
