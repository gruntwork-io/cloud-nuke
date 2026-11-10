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

// ArtifactRegistryAPI is the part of the Artifact Registry API this resource
// uses. The SDK answers each list with an iterator a test cannot build, so the
// iteration lives in the adapter below and the decisions stay testable.
type ArtifactRegistryAPI interface {
	// ListLocations returns the location ids the project can hold
	// repositories in. The API rejects the "-" wildcard other GCP APIs
	// accept, which is why the locations are listed first.
	ListLocations(ctx context.Context, project string) ([]string, error)

	// ListRepositories returns the repositories in one location.
	ListRepositories(ctx context.Context, parent string) ([]ArtifactRepository, error)

	// DeleteRepository deletes one repository and waits for the operation to
	// finish. Deleting one that is already gone is not an error.
	DeleteRepository(ctx context.Context, name string) error
}

// ArtifactRepository is what this resource needs about one repository.
type ArtifactRepository struct {
	// Name is the fully qualified name, which carries the location.
	Name    string
	Created time.Time
	Labels  map[string]string
}

// NewArtifactRegistryRepositories creates a new Artifact Registry resource using the generic resource pattern.
func NewArtifactRegistryRepositories() GcpResource {
	return NewGcpResource(&resource.Resource[ArtifactRegistryAPI]{
		ResourceTypeName: "artifact-registry",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[ArtifactRegistryAPI], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := artifactregistry.NewClient(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Artifact Registry client: %w", err)
				return
			}
			r.Client = artifactRegistryClient{client: client}
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ArtifactRegistry
		},
		Lister: listArtifactRegistryRepositories,
		Nuker:  resource.SequentialDeleter(deleteArtifactRegistryRepository),
	})
}

// listArtifactRegistryRepositories retrieves all Artifact Registry repositories
// in the project that match the config filters, one location at a time.
func listArtifactRegistryRepositories(ctx context.Context, client ArtifactRegistryAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	locations, err := client.ListLocations(ctx, scope.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("error listing artifact registry locations: %w", err)
	}

	var result []*string
	for _, location := range locations {
		parent := fmt.Sprintf("projects/%s/locations/%s", scope.ProjectID, location)

		repositories, err := client.ListRepositories(ctx, parent)
		if err != nil {
			// One location answering an error is not worth ending the scan:
			// a project can hold repositories in the others.
			logging.Debugf("error listing artifact registry repositories in %s, skipping location: %v", location, err)
			continue
		}

		for _, repository := range repositories {
			fullName := repository.Name

			// Config filters match the short name, as they do for a GCS
			// bucket, while the delete needs the fully qualified name.
			shortName := fullName[strings.LastIndex(fullName, "/")+1:]

			// A repository GCP created during the Container Registry migration
			// stays: deleting it breaks gcr.io image pulls for the project.
			if _, isGCRManaged := gcrManagedRepos[shortName]; isGCRManaged {
				logging.Debugf("Skipping GCR-managed Artifact Registry repository: %s", fullName)
				continue
			}

			created := repository.Created

			// An empty map rather than nil tells the config layer this
			// resource reports labels.
			labels := repository.Labels
			if labels == nil {
				labels = map[string]string{}
			}

			if cfg.ShouldInclude(config.ResourceValue{Name: &shortName, Time: &created, Tags: labels}) {
				result = append(result, &fullName)
			}
		}
	}

	return result, nil
}

// deleteArtifactRegistryRepository deletes a single Artifact Registry repository.
func deleteArtifactRegistryRepository(ctx context.Context, client ArtifactRegistryAPI, name *string) error {
	if err := client.DeleteRepository(ctx, *name); err != nil {
		return fmt.Errorf("error deleting artifact registry repository %s: %w", *name, err)
	}

	logging.Debugf("Deleted Artifact Registry repository: %s", *name)
	return nil
}

// artifactRegistryClient drives the SDK: the two iterators, the long-running
// operation and the rate-limiting delay.
type artifactRegistryClient struct {
	client *artifactregistry.Client
}

// artifactRegistryDeleteDelay keeps deletions inside the 600 requests a minute
// quota; op.Wait already accounts for how long a delete takes.
const artifactRegistryDeleteDelay = 500 * time.Millisecond

func (c artifactRegistryClient) ListLocations(ctx context.Context, project string) ([]string, error) {
	var locations []string

	it := c.client.ListLocations(ctx, &locationpb.ListLocationsRequest{
		Name: fmt.Sprintf("projects/%s", project),
	})
	for {
		location, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return locations, nil
		}
		if err != nil {
			return nil, err
		}
		locations = append(locations, location.LocationId)
	}
}

func (c artifactRegistryClient) ListRepositories(ctx context.Context, parent string) ([]ArtifactRepository, error) {
	var found []ArtifactRepository

	it := c.client.ListRepositories(ctx, &artifactregistrypb.ListRepositoriesRequest{Parent: parent})
	for {
		repo, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return found, nil
		}
		if err != nil {
			return nil, err
		}

		repository := ArtifactRepository{Name: repo.Name, Labels: repo.GetLabels()}
		if repo.CreateTime != nil {
			repository.Created = repo.CreateTime.AsTime()
		}
		found = append(found, repository)
	}
}

func (c artifactRegistryClient) DeleteRepository(ctx context.Context, name string) error {
	op, err := c.client.DeleteRepository(ctx, &artifactregistrypb.DeleteRepositoryRequest{Name: name})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Artifact Registry repository %s already deleted, skipping", name)
			return nil
		}
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("waiting for deletion: %w", err)
	}

	select {
	case <-time.After(artifactRegistryDeleteDelay):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

var _ ArtifactRegistryAPI = artifactRegistryClient{}
