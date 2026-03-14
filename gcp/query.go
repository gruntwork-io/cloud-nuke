package gcp

import (
	"context"
	"fmt"
	"time"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/gruntwork-io/go-commons/collections"
)

// Query represents the desired parameters for scanning GCP resources.
// This mirrors the aws.Query struct for interface consistency.
type Query struct {
	ProjectID            string
	ResourceTypes        []string
	ExcludeResourceTypes []string
	Regions              []string
	ExcludeRegions       []string
	ExcludeAfter         *time.Time
	IncludeAfter         *time.Time
	Timeout              *time.Duration
	ExcludeFirstSeen     bool
}

// Validate ensures the query has valid defaults and that the project is accessible.
// If no regions are specified, it defaults to GlobalRegion.
// ExcludeRegions are filtered out from the region list.
func (q *Query) Validate() error {
	if q.ProjectID == "" {
		return fmt.Errorf("--project-id is required")
	}

	if err := validateProjectID(q.ProjectID, q.Timeout); err != nil {
		return err
	}

	return q.validateRegions()
}

func (q *Query) validateRegions() error {
	if len(q.Regions) == 0 {
		q.Regions = []string{GlobalRegion}
	}

	if len(q.ExcludeRegions) > 0 {
		var filtered []string
		for _, region := range q.Regions {
			if !collections.ListContainsElement(q.ExcludeRegions, region) {
				filtered = append(filtered, region)
			}
		}
		q.Regions = filtered
	}

	if len(q.Regions) == 0 {
		return fmt.Errorf("no regions to process after applying exclusions")
	}

	return nil
}

// validateProjectID verifies that the GCP project exists and is accessible
// with the current credentials using the Cloud Resource Manager API.
func validateProjectID(projectID string, timeout *time.Duration) error {
	t := 30 * time.Second
	if timeout != nil {
		t = *timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Resource Manager client: %w", err)
	}
	defer client.Close()

	_, err = client.GetProject(ctx, &resourcemanagerpb.GetProjectRequest{
		Name: "projects/" + projectID,
	})
	if err != nil {
		return fmt.Errorf("invalid or inaccessible project %q: %w", projectID, err)
	}

	return nil
}
