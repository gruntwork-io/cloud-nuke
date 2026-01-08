package reporting

import (
	"context"
	"sync"
)

// Renderer processes events and produces output
type Renderer interface {
	// OnEvent is called for each event (for streaming output)
	OnEvent(event Event)

	// Render is called at the end to produce final output
	Render() error
}

// Collector receives events and routes them to renderers.
// All event emission is serialized, so renderers don't need their own synchronization.
type Collector struct {
	mu        sync.Mutex
	renderers []Renderer
	closed    bool
}

// NewCollector creates a new Collector
func NewCollector() *Collector {
	return &Collector{
		renderers: make([]Renderer, 0),
	}
}

// AddRenderer adds a renderer to receive events
func (c *Collector) AddRenderer(r Renderer) {
	if r == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.renderers = append(c.renderers, r)
}

// RecordFound records a discovered resource
func (c *Collector) RecordFound(resourceType, region, identifier string, nukable bool, reason string) {
	c.emit(ResourceFound{
		ResourceType: resourceType,
		Region:       region,
		Identifier:   identifier,
		Nukable:      nukable,
		Reason:       reason,
	})
}

// RecordDeleted records a deletion result
func (c *Collector) RecordDeleted(resourceType, region, identifier string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	c.emit(ResourceDeleted{
		ResourceType: resourceType,
		Region:       region,
		Identifier:   identifier,
		Success:      err == nil,
		Error:        errStr,
	})
}

// RecordError records a general error
func (c *Collector) RecordError(resourceType, description string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	c.emit(GeneralError{
		ResourceType: resourceType,
		Description:  description,
		Error:        errStr,
	})
}

// emit sends an event to all renderers
func (c *Collector) emit(event Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	for _, r := range c.renderers {
		r.OnEvent(event)
	}
}

// Complete marks collection as finished and renders final output.
// Safe to call multiple times - subsequent calls are no-ops.
func (c *Collector) Complete() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already completed
	}
	c.closed = true
	renderers := c.renderers
	c.mu.Unlock()

	for _, r := range renderers {
		if err := r.Render(); err != nil {
			return err
		}
	}

	return nil
}

// Context key for collector
type collectorKey struct{}

// WithCollector adds a collector to the context
func WithCollector(ctx context.Context, c *Collector) context.Context {
	return context.WithValue(ctx, collectorKey{}, c)
}

// FromContext retrieves the collector from context (nil if not present)
func FromContext(ctx context.Context) *Collector {
	if ctx == nil {
		return nil
	}
	if c, ok := ctx.Value(collectorKey{}).(*Collector); ok {
		return c
	}
	return nil
}
