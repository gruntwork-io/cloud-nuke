package mcp

import (
	"sync"

	"github.com/gruntwork-io/cloud-nuke/reporting"
)

// MCPRenderer implements reporting.Renderer by accumulating events in memory.
// After the operation completes, the tool handler reads the accumulated state.
type MCPRenderer struct {
	mu      sync.Mutex
	found   []reporting.ResourceFound
	deleted []reporting.ResourceDeleted
	errors  []reporting.GeneralError
}

// newMCPRenderer creates a new MCPRenderer.
func newMCPRenderer() *MCPRenderer {
	return &MCPRenderer{}
}

// OnEvent collects events for later serialization.
// Events not relevant to the final result (ScanProgress, NukeProgress, etc.) are ignored.
func (r *MCPRenderer) OnEvent(event reporting.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch e := event.(type) {
	case reporting.ResourceFound:
		r.found = append(r.found, e)
	case reporting.ResourceDeleted:
		r.deleted = append(r.deleted, e)
	case reporting.GeneralError:
		r.errors = append(r.errors, e)
	}
}

// buildResourceInfos converts accumulated ResourceFound events into ResourceInfo slices.
func (r *MCPRenderer) buildResourceInfos() []ResourceInfo {
	resources := make([]ResourceInfo, 0, len(r.found))
	for _, e := range r.found {
		resources = append(resources, ResourceInfo{
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Identifier:   e.Identifier,
			Nukable:      e.Nukable,
			Reason:       e.Reason,
		})
	}
	return resources
}

// buildErrorInfos converts accumulated GeneralError events into GeneralErrorInfo slices.
func (r *MCPRenderer) buildErrorInfos() []GeneralErrorInfo {
	errors := make([]GeneralErrorInfo, 0, len(r.errors))
	for _, e := range r.errors {
		errors = append(errors, GeneralErrorInfo{
			ResourceType: e.ResourceType,
			Description:  e.Description,
			Error:        e.Error,
		})
	}
	return errors
}

// BuildInspectResult converts accumulated state into an InspectResult.
func (r *MCPRenderer) BuildInspectResult() InspectResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	byType := make(map[string]int)
	byRegion := make(map[string]int)
	nukableCount := 0
	nonNukableCount := 0

	resources := r.buildResourceInfos()
	for _, res := range resources {
		byType[res.ResourceType]++
		byRegion[res.Region]++
		if res.Nukable {
			nukableCount++
		} else {
			nonNukableCount++
		}
	}

	return InspectResult{
		Resources: resources,
		Errors:    r.buildErrorInfos(),
		Summary: InspectSummary{
			TotalResources: len(r.found),
			Nukable:        nukableCount,
			NonNukable:     nonNukableCount,
			GeneralErrors:  len(r.errors),
			ByType:         byType,
			ByRegion:       byRegion,
		},
	}
}

// BuildNukeResult converts accumulated state into a NukeResult.
func (r *MCPRenderer) BuildNukeResult(dryRun bool) NukeResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	deleted := make([]DeletedInfo, 0, len(r.deleted))
	deletedCount := 0
	failedCount := 0
	for _, e := range r.deleted {
		status := "deleted"
		if !e.Success {
			status = "failed"
			failedCount++
		} else {
			deletedCount++
		}
		deleted = append(deleted, DeletedInfo{
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Identifier:   e.Identifier,
			Status:       status,
			Error:        e.Error,
		})
	}

	return NukeResult{
		DryRun:    dryRun,
		Resources: r.buildResourceInfos(),
		Deleted:   deleted,
		Errors:    r.buildErrorInfos(),
		Summary: NukeSummary{
			Found:         len(r.found),
			Deleted:       deletedCount,
			Failed:        failedCount,
			GeneralErrors: len(r.errors),
		},
	}
}
