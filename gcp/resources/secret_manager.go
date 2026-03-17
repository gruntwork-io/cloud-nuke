package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SecretManagerClients holds clients for the global and regional Secret Manager endpoints.
type SecretManagerClients struct {
	Global   *secretmanager.Client
	Regional map[string]*secretmanager.Client // location -> client
}

// NewSecretManagerSecrets creates a new Secret Manager resource.
func NewSecretManagerSecrets() GcpResource {
	return NewGcpResource(&resource.Resource[*SecretManagerClients]{
		ResourceTypeName: "gcp-secret-manager",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*SecretManagerClients], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			r.Scope.Locations = cfg.Locations
			r.Scope.ExcludeLocations = cfg.ExcludeLocations

			clients, err := initSecretManagerClients(cfg)
			if err != nil {
				// Panic is recovered by GcpResourceAdapter.Init() and stored as InitializationError,
				// causing subsequent GetAndSetIdentifiers/Nuke calls to return the error gracefully.
				panic(fmt.Sprintf("failed to create Secret Manager clients: %v", err))
			}
			r.Client = clients
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpSecretManager
		},
		Lister: listSecretManagerSecrets,
		Nuker:  resource.SequentialDeleter(deleteSecretManagerSecret),
	})
}

// initSecretManagerClients creates the appropriate Secret Manager clients based on location filters.
// A global client is always created first to discover available regional endpoints via ListLocations,
// then regional clients are created for each location that passes the filter.
func initSecretManagerClients(cfg GcpConfig) (*SecretManagerClients, error) {
	ctx := context.Background()
	clients := &SecretManagerClients{
		Regional: make(map[string]*secretmanager.Client),
	}

	// Always create the global client — needed for ListLocations discovery and global secrets.
	globalClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create global Secret Manager client: %w", err)
	}

	// Determine whether the user wants global secrets included.
	wantGlobal := shouldIncludeGlobalEndpoint(cfg.Locations, cfg.ExcludeLocations)
	if wantGlobal {
		clients.Global = globalClient
	}

	// Discover available regional endpoints via ListLocations on the global client.
	regionalLocations, err := discoverSecretManagerLocations(ctx, globalClient, cfg)

	// Close the global client if it's only used for discovery, not for querying secrets.
	if !wantGlobal {
		_ = globalClient.Close()
	}
	if err != nil {
		logging.Debugf("Failed to discover Secret Manager locations: %v", err)
		// Fall back to global-only if discovery fails
		if clients.Global == nil {
			return nil, fmt.Errorf("failed to discover Secret Manager locations and global endpoint not requested")
		}
		return clients, nil
	}

	// Create a regional client for each discovered location.
	// If a client fails to create, secrets in that location won't be discovered
	// (listing is per-client), so there's no list/delete mismatch.
	for _, loc := range regionalLocations {
		endpoint := fmt.Sprintf("secretmanager.%s.rep.googleapis.com:443", loc)
		client, err := secretmanager.NewClient(ctx, option.WithEndpoint(endpoint))
		if err != nil {
			logging.Debugf("Failed to create regional Secret Manager client for %s: %v", loc, err)
			continue
		}
		clients.Regional[loc] = client
	}

	if clients.Global == nil && len(clients.Regional) == 0 {
		return nil, fmt.Errorf("failed to create any Secret Manager clients")
	}

	return clients, nil
}

// discoverSecretManagerLocations uses the ListLocations RPC on the global client to
// discover available regional endpoints, then filters them by the location config.
func discoverSecretManagerLocations(ctx context.Context, client *secretmanager.Client, cfg GcpConfig) ([]string, error) {
	locReq := &locationpb.ListLocationsRequest{
		Name: fmt.Sprintf("projects/%s", cfg.ProjectID),
	}

	var locations []string
	it := client.ListLocations(ctx, locReq)
	for {
		loc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing Secret Manager locations: %w", err)
		}

		if !MatchesLocationFilter(loc.LocationId, cfg.Locations, cfg.ExcludeLocations) {
			continue
		}
		// Skip "global" — it's handled by the global client, not a regional endpoint.
		if strings.EqualFold(loc.LocationId, "global") {
			continue
		}
		locations = append(locations, loc.LocationId)
	}

	return locations, nil
}

func isExcluded(location string, excludeLocations []string) bool {
	for _, exc := range excludeLocations {
		if strings.EqualFold(exc, location) {
			return true
		}
	}
	return false
}

// shouldIncludeGlobalEndpoint determines whether the global Secret Manager endpoint
// should be queried based on the location filters.
//
// No locations specified → include global (unless explicitly excluded)
// Locations specified → include global only if "global" is in the list (and not excluded)
func shouldIncludeGlobalEndpoint(locations []string, excludeLocations []string) bool {
	if isExcluded("global", excludeLocations) {
		return false
	}
	if len(locations) == 0 {
		return true
	}
	for _, loc := range locations {
		if strings.EqualFold(loc, "global") {
			return true
		}
	}
	return false
}

// listSecretManagerSecrets retrieves all secrets from the configured endpoints.
func listSecretManagerSecrets(ctx context.Context, clients *SecretManagerClients, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string
	var listErrors int
	totalEndpoints := len(clients.Regional)
	if clients.Global != nil {
		totalEndpoints++
	}

	// List secrets from the global endpoint (parent: projects/{project})
	if clients.Global != nil {
		parent := fmt.Sprintf("projects/%s", scope.ProjectID)
		secrets, err := listSecretsFromClient(ctx, clients.Global, parent, cfg)
		if err != nil {
			logging.Debugf("Error listing global secrets: %v", err)
			listErrors++
		} else {
			result = append(result, secrets...)
		}
	}

	// List secrets from each regional endpoint (parent: projects/{project}/locations/{location})
	for location, client := range clients.Regional {
		parent := fmt.Sprintf("projects/%s/locations/%s", scope.ProjectID, location)
		secrets, err := listSecretsFromClient(ctx, client, parent, cfg)
		if err != nil {
			logging.Debugf("Error listing secrets in location %s: %v", location, err)
			listErrors++
			continue
		}
		result = append(result, secrets...)
	}

	if listErrors > 0 && listErrors == totalEndpoints {
		logging.Warnf("All %d Secret Manager endpoints failed to list secrets", totalEndpoints)
	}

	return result, nil
}

// listSecretsFromClient lists secrets from a single Secret Manager client.
// Parent is "projects/{project}" for global or "projects/{project}/locations/{location}" for regional.
func listSecretsFromClient(ctx context.Context, client *secretmanager.Client, parent string, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	req := &secretmanagerpb.ListSecretsRequest{
		Parent: parent,
	}

	it := client.ListSecrets(ctx, req)
	for {
		secret, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing secrets: %w", err)
		}

		// secret.Name is fully qualified: projects/{project}/secrets/{secret}
		// For regional secrets: projects/{project}/locations/{location}/secrets/{secret}
		fullName := secret.Name

		// Extract short name for config filtering
		shortName := fullName[strings.LastIndex(fullName, "/")+1:]

		// Use CreateTime for time-based filtering
		var resourceTime time.Time
		if secret.CreateTime != nil {
			resourceTime = secret.CreateTime.AsTime()
		}

		// Pass labels for tag-based filtering
		labels := secret.GetLabels()
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

	return result, nil
}

// Rate limiting delay between secret deletions
const secretManagerDeleteDelay = 500 * time.Millisecond

// deleteSecretManagerSecret deletes a single secret.
// It determines the correct client (global vs regional) from the fully qualified secret name.
func deleteSecretManagerSecret(ctx context.Context, clients *SecretManagerClients, name *string) error {
	secretName := *name

	// Determine which client to use based on the secret name format.
	// Regional: projects/{project}/locations/{location}/secrets/{secret}
	// Global:   projects/{project}/secrets/{secret}
	client := pickClientForSecret(clients, secretName)
	if client == nil {
		return fmt.Errorf("no client available for secret %s", secretName)
	}

	req := &secretmanagerpb.DeleteSecretRequest{
		Name: secretName,
	}

	if err := client.DeleteSecret(ctx, req); err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Secret %s already deleted, skipping", secretName)
			return nil
		}
		return fmt.Errorf("error deleting secret %s: %w", secretName, err)
	}

	logging.Debugf("Deleted Secret Manager secret: %s", secretName)

	select {
	case <-time.After(secretManagerDeleteDelay):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// pickClientForSecret returns the appropriate client for the given secret name.
func pickClientForSecret(clients *SecretManagerClients, secretName string) *secretmanager.Client {
	// Parse location from name: projects/{project}/locations/{location}/secrets/{secret}
	location := ExtractLocationFromResourceName(secretName)
	if location != "" {
		if client, ok := clients.Regional[location]; ok {
			return client
		}
		return nil
	}
	// Global secret: projects/{project}/secrets/{secret}
	return clients.Global
}
