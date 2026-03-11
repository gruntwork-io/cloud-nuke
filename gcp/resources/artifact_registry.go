package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/iterator"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gcrManagedRepos are repositories automatically created by GCP during Container Registry to
// Artifact Registry migration. Deleting these breaks gcr.io image pulls for the project.
var gcrManagedRepos = map[string]struct{}{
	"gcr.io":      {},
	"us.gcr.io":   {},
	"eu.gcr.io":   {},
	"asia.gcr.io": {},
}

// NewArtifactRegistryRepositories creates a new Artifact Registry resource using the generic resource pattern.
func NewArtifactRegistryRepositories() GcpResource {
	return NewGcpResource(&resource.Resource[*artifactregistry.Client]{
		ResourceTypeName: "artifact-registry",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*artifactregistry.Client], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := artifactregistry.NewClient(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Artifact Registry client: %w", err)
				return
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ArtifactRegistry
		},
		Lister: listArtifactRegistryRepositories,
		Nuker:  resource.SequentialDeleter(deleteArtifactRegistryRepository),
	})
}

// listArtifactRegistryRepositories retrieves all Artifact Registry repositories in the project that match the config filters.
// Unlike Cloud Functions, the Artifact Registry API does not support the wildcard "locations/-",
// so we first list all available locations and then query repositories in each one.
func listArtifactRegistryRepositories(ctx context.Context, client *artifactregistry.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	// First, list all available locations for this project
	locReq := &locationpb.ListLocationsRequest{
		Name: fmt.Sprintf("projects/%s", scope.ProjectID),
	}

	locIt := client.ListLocations(ctx, locReq)
	for {
		loc, err := locIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing artifact registry locations: %w", err)
		}

		// List repositories in this location
		parent := fmt.Sprintf("projects/%s/locations/%s", scope.ProjectID, loc.LocationId)
		req := &artifactregistrypb.ListRepositoriesRequest{
			Parent: parent,
		}

		it := client.ListRepositories(ctx, req)
		for {
			repo, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				logging.Debugf("error listing artifact registry repositories in %s, skipping location: %v", loc.LocationId, err)
				break
			}

			// repo.Name is the fully qualified name: projects/{project}/locations/{location}/repositories/{repository}
			fullName := repo.Name

			// Extract just the repository short name for config filtering,
			// consistent with GCS buckets which filter on the short name.
			// The full path is retained as the delete identifier since the API requires it.
			shortName := fullName[strings.LastIndex(fullName, "/")+1:]

			// Skip GCR-managed repositories created automatically during Container Registry
			// to Artifact Registry migration. Deleting these breaks gcr.io image pulls.
			if _, isGCRManaged := gcrManagedRepos[shortName]; isGCRManaged {
				logging.Debugf("Skipping GCR-managed Artifact Registry repository: %s", fullName)
				continue
			}

			// Use CreateTime as the resource timestamp for time-based filtering
			var resourceTime time.Time
			if repo.CreateTime != nil {
				resourceTime = repo.CreateTime.AsTime()
			}

			// Pass labels for tag-based filtering.
			// Use an empty map (not nil) to signal the resource supports labels.
			labels := repo.GetLabels()
			if labels == nil {
				labels = map[string]string{}
			}

			resourceValue := config.ResourceValue{
				Name: &shortName,
				Time: &resourceTime,
				Tags: labels,
			}

			if cfg.ShouldInclude(resourceValue) {
				result = append(result, &fullName)
			}
		}
	}

	return result, nil
}

// Rate limiting delay between repository deletions to avoid API quota issues.
// 500ms is sufficient given the 600 req/min quota; op.Wait already accounts for delete duration.
const artifactRegistryDeleteDelay = 500 * time.Millisecond

// deleteArtifactRegistryRepository deletes a single Artifact Registry repository.
func deleteArtifactRegistryRepository(ctx context.Context, client *artifactregistry.Client, name *string) error {
	repoName := *name

	req := &artifactregistrypb.DeleteRepositoryRequest{
		Name: repoName,
	}

	op, err := client.DeleteRepository(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Artifact Registry repository %s already deleted, skipping", repoName)
			return nil
		}
		return fmt.Errorf("error deleting artifact registry repository %s: %w", repoName, err)
	}

	// Wait for the long-running delete operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for artifact registry repository %s deletion: %w", repoName, err)
	}

	logging.Debugf("Deleted Artifact Registry repository: %s", repoName)

	// Rate limiting delay to avoid API quota issues.
	// Uses select so the delay is cancelled if the context is cancelled.
	select {
	case <-time.After(artifactRegistryDeleteDelay):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
