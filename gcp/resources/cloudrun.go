package resources

import (
	"context"
	"fmt"

	runv1 "google.golang.org/api/run/v1"
)

// listCloudRunLocations returns all Cloud Run locations available for the given project
// using the Cloud Run Admin API v1 REST client.
func listCloudRunLocations(ctx context.Context, projectID string) ([]string, error) {
	svc, err := runv1.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Run v1 service: %w", err)
	}

	var locations []string
	parent := fmt.Sprintf("projects/%s", projectID)

	err = svc.Projects.Locations.List(parent).Pages(ctx, func(page *runv1.ListLocationsResponse) error {
		for _, loc := range page.Locations {
			locations = append(locations, loc.LocationId)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error listing Cloud Run locations: %w", err)
	}

	return locations, nil
}
