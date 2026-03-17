package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
)

// Query represents the desired parameters for scanning GCP resources.
// This mirrors the aws.Query struct for interface consistency.
type Query struct {
	ProjectID            string
	ResourceTypes        []string
	ExcludeResourceTypes []string
	Locations            []string
	ExcludeLocations     []string
	Timeout              *time.Duration
	ExcludeFirstSeen     bool
}

// Validate ensures the query has valid defaults and that the project is accessible.
// An empty Locations means "all locations" — each resource decides its broadest scope.
// ExcludeLocations are filtered out from the location list if locations are specified.
func (q *Query) Validate(ctx context.Context) error {
	if q.ProjectID == "" {
		return fmt.Errorf("--project-id is required")
	}

	if err := validateProjectID(ctx, q.ProjectID, q.Timeout); err != nil {
		return err
	}

	return q.validateLocations()
}

func (q *Query) validateLocations() error {
	if len(q.ExcludeLocations) > 0 && len(q.Locations) > 0 {
		var filtered []string
		for _, loc := range q.Locations {
			if !containsIgnoreCase(q.ExcludeLocations, loc) {
				filtered = append(filtered, loc)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("all specified --region values were excluded by --exclude-region; nothing to scan")
		}
		q.Locations = filtered
	}

	return nil
}

// validateProjectID verifies that the GCP project exists and is accessible
// with the current credentials using the Cloud Resource Manager API.
func validateProjectID(ctx context.Context, projectID string, timeout *time.Duration) error {
	t := 30 * time.Second
	if timeout != nil {
		t = *timeout
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Resource Manager client: %w", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.GetProject(ctx, &resourcemanagerpb.GetProjectRequest{
		Name: "projects/" + projectID,
	})
	if err != nil {
		return fmt.Errorf("invalid or inaccessible project %q: %w", projectID, err)
	}

	return nil
}

func containsIgnoreCase(list []string, target string) bool {
	for _, s := range list {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}
