package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	// sageMakerMaxWaitTime is the maximum time to wait for any resource deletion
	sageMakerMaxWaitTime = 30 * time.Minute
	// sageMakerRetryDelay is the delay between retry attempts
	sageMakerRetryDelay = 10 * time.Second
)

// SageMakerStudioAPI defines the interface for SageMaker Studio operations.
type SageMakerStudioAPI interface {
	// Domain operations
	ListDomains(ctx context.Context, params *sagemaker.ListDomainsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListDomainsOutput, error)
	DescribeDomain(ctx context.Context, params *sagemaker.DescribeDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeDomainOutput, error)
	DeleteDomain(ctx context.Context, params *sagemaker.DeleteDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteDomainOutput, error)

	// UserProfile operations
	ListUserProfiles(ctx context.Context, params *sagemaker.ListUserProfilesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListUserProfilesOutput, error)
	DeleteUserProfile(ctx context.Context, params *sagemaker.DeleteUserProfileInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteUserProfileOutput, error)

	// App operations
	ListApps(ctx context.Context, params *sagemaker.ListAppsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListAppsOutput, error)
	DeleteApp(ctx context.Context, params *sagemaker.DeleteAppInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteAppOutput, error)

	// Space operations
	ListSpaces(ctx context.Context, params *sagemaker.ListSpacesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListSpacesOutput, error)
	DeleteSpace(ctx context.Context, params *sagemaker.DeleteSpaceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteSpaceOutput, error)

	// MLflow tracking server operations
	ListMlflowTrackingServers(ctx context.Context, params *sagemaker.ListMlflowTrackingServersInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListMlflowTrackingServersOutput, error)
	DeleteMlflowTrackingServer(ctx context.Context, params *sagemaker.DeleteMlflowTrackingServerInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteMlflowTrackingServerOutput, error)
}

// NewSageMakerStudio creates a new SageMaker Studio resource using the generic resource pattern.
func NewSageMakerStudio() AwsResource {
	return NewAwsResource(&resource.Resource[SageMakerStudioAPI]{
		ResourceTypeName: "sagemaker-studio",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SageMakerStudioAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = sagemaker.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return config.ResourceType{
				Timeout:            c.SageMakerStudioDomain.Timeout,
				ProtectUntilExpire: c.SageMakerStudioDomain.ProtectUntilExpire,
			}
		},
		Lister: listSageMakerDomains,
		Nuker:  nukeSageMakerDomains,
	})
}

// listSageMakerDomains retrieves all SageMaker Studio domains that match the config filters.
func listSageMakerDomains(ctx context.Context, client SageMakerStudioAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var domainIDs []*string

	paginator := sagemaker.NewListDomainsPaginator(client, &sagemaker.ListDomainsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, domain := range page.Domains {
			if domain.DomainId != nil {
				logging.Debugf("Found SageMaker Studio domain: %s (Status: %s)", *domain.DomainId, domain.Status)
				domainIDs = append(domainIDs, domain.DomainId)
			}
		}
	}

	return domainIDs, nil
}

// nukeSageMakerDomains is a custom nuker function for SageMaker Studio domains.
// SageMaker domains require deleting sub-resources (apps, spaces, user profiles, MLflow servers) before deletion.
func nukeSageMakerDomains(ctx context.Context, client SageMakerStudioAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	if len(identifiers) == 0 {
		logging.Debugf("No SageMaker Studio domains to nuke in %s", scope)
		return nil
	}

	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	// Process domains concurrently
	wg := new(sync.WaitGroup)
	resultsChan := make(chan resource.NukeResult, len(identifiers))

	for _, domainID := range identifiers {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			err := nukeSageMakerDomain(ctx, client, scope, id)
			resultsChan <- resource.NukeResult{Identifier: id, Error: err}
		}(aws.ToString(domainID))
	}

	wg.Wait()
	close(resultsChan)

	// Collect results
	results := make([]resource.NukeResult, 0, len(identifiers))
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// nukeSageMakerDomain deletes a single SageMaker Studio domain and all its resources.
func nukeSageMakerDomain(ctx context.Context, client SageMakerStudioAPI, scope resource.Scope, domainID string) error {
	logging.Debugf("Starting deletion of SageMaker Studio domain %s in %s", domainID, scope)

	// Phase 1: Delete apps, spaces, and MLflow tracking servers in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(3)
	go func() {
		defer wg.Done()
		if err := deleteAllApps(ctx, client, domainID, scope); err != nil {
			errChan <- err
		}
	}()
	go func() {
		defer wg.Done()
		if err := deleteAllSpaces(ctx, client, domainID, scope); err != nil {
			errChan <- err
		}
	}()
	go func() {
		defer wg.Done()
		if err := deleteAllMlflowServers(ctx, client, scope); err != nil {
			errChan <- err
		}
	}()
	wg.Wait()
	close(errChan)

	// Check for errors from phase 1
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Phase 2: Delete user profiles (after apps are deleted)
	if err := deleteAllUserProfiles(ctx, client, domainID, scope); err != nil {
		return err
	}

	// Phase 3: Delete the domain itself
	if _, err := client.DeleteDomain(ctx, &sagemaker.DeleteDomainInput{
		DomainId:        aws.String(domainID),
		RetentionPolicy: &types.RetentionPolicy{HomeEfsFileSystem: types.RetentionTypeDelete},
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	// Wait for domain deletion
	return waitForDeletion(ctx, client, fmt.Sprintf("domain %s", domainID), func() (bool, error) {
		_, err := client.DescribeDomain(ctx, &sagemaker.DescribeDomainInput{DomainId: aws.String(domainID)})
		return err == nil, nil
	})
}

// deleteAllApps deletes all apps for a domain.
func deleteAllApps(ctx context.Context, client SageMakerStudioAPI, domainID string, scope resource.Scope) error {
	apps, err := client.ListApps(ctx, &sagemaker.ListAppsInput{DomainIdEquals: aws.String(domainID)})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(apps.Apps))

	for _, app := range apps.Apps {
		if app.Status == types.AppStatusDeleted {
			continue
		}
		wg.Add(1)
		go func(a types.AppDetails) {
			defer wg.Done()
			if err := deleteApp(ctx, client, domainID, a); err != nil {
				errChan <- err
			}
		}(app)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// deleteApp deletes a single app and waits for deletion.
func deleteApp(ctx context.Context, client SageMakerStudioAPI, domainID string, app types.AppDetails) error {
	input := &sagemaker.DeleteAppInput{
		AppName:         app.AppName,
		AppType:         app.AppType,
		DomainId:        aws.String(domainID),
		UserProfileName: app.UserProfileName,
		SpaceName:       app.SpaceName,
	}

	if _, err := client.DeleteApp(ctx, input); err != nil {
		return errors.WithStackTrace(err)
	}

	return waitForDeletion(ctx, client, fmt.Sprintf("app %s", aws.ToString(app.AppName)), func() (bool, error) {
		apps, err := client.ListApps(ctx, &sagemaker.ListAppsInput{DomainIdEquals: aws.String(domainID)})
		if err != nil {
			return false, err
		}
		for _, a := range apps.Apps {
			if aws.ToString(a.AppName) == aws.ToString(app.AppName) && a.Status != types.AppStatusDeleted {
				return true, nil
			}
		}
		return false, nil
	})
}

// deleteAllSpaces deletes all spaces for a domain.
func deleteAllSpaces(ctx context.Context, client SageMakerStudioAPI, domainID string, scope resource.Scope) error {
	spaces, err := client.ListSpaces(ctx, &sagemaker.ListSpacesInput{DomainIdEquals: aws.String(domainID)})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(spaces.Spaces))

	for _, space := range spaces.Spaces {
		wg.Add(1)
		go func(s types.SpaceDetails) {
			defer wg.Done()
			if err := deleteSpace(ctx, client, domainID, s); err != nil {
				errChan <- err
			}
		}(space)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// deleteSpace deletes a single space and waits for deletion.
func deleteSpace(ctx context.Context, client SageMakerStudioAPI, domainID string, space types.SpaceDetails) error {
	if _, err := client.DeleteSpace(ctx, &sagemaker.DeleteSpaceInput{
		DomainId:  aws.String(domainID),
		SpaceName: space.SpaceName,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	return waitForDeletion(ctx, client, fmt.Sprintf("space %s", aws.ToString(space.SpaceName)), func() (bool, error) {
		spaces, err := client.ListSpaces(ctx, &sagemaker.ListSpacesInput{DomainIdEquals: aws.String(domainID)})
		if err != nil {
			return false, err
		}
		for _, s := range spaces.Spaces {
			if aws.ToString(s.SpaceName) == aws.ToString(space.SpaceName) {
				return true, nil
			}
		}
		return false, nil
	})
}

// deleteAllMlflowServers deletes all MLflow tracking servers.
func deleteAllMlflowServers(ctx context.Context, client SageMakerStudioAPI, scope resource.Scope) error {
	servers, err := client.ListMlflowTrackingServers(ctx, &sagemaker.ListMlflowTrackingServersInput{})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(servers.TrackingServerSummaries))

	for _, server := range servers.TrackingServerSummaries {
		wg.Add(1)
		go func(s types.TrackingServerSummary) {
			defer wg.Done()
			if err := deleteMlflowServer(ctx, client, s); err != nil {
				errChan <- err
			}
		}(server)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// deleteMlflowServer deletes a single MLflow tracking server and waits for deletion.
func deleteMlflowServer(ctx context.Context, client SageMakerStudioAPI, server types.TrackingServerSummary) error {
	if _, err := client.DeleteMlflowTrackingServer(ctx, &sagemaker.DeleteMlflowTrackingServerInput{
		TrackingServerName: server.TrackingServerName,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	return waitForDeletion(ctx, client, fmt.Sprintf("MLflow server %s", aws.ToString(server.TrackingServerName)), func() (bool, error) {
		servers, err := client.ListMlflowTrackingServers(ctx, &sagemaker.ListMlflowTrackingServersInput{})
		if err != nil {
			return false, err
		}
		for _, s := range servers.TrackingServerSummaries {
			if aws.ToString(s.TrackingServerName) == aws.ToString(server.TrackingServerName) {
				return true, nil
			}
		}
		return false, nil
	})
}

// deleteAllUserProfiles deletes all user profiles for a domain.
func deleteAllUserProfiles(ctx context.Context, client SageMakerStudioAPI, domainID string, scope resource.Scope) error {
	profiles, err := client.ListUserProfiles(ctx, &sagemaker.ListUserProfilesInput{DomainIdEquals: aws.String(domainID)})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(profiles.UserProfiles))

	for _, profile := range profiles.UserProfiles {
		wg.Add(1)
		go func(p types.UserProfileDetails) {
			defer wg.Done()
			if err := deleteUserProfile(ctx, client, domainID, p); err != nil {
				errChan <- err
			}
		}(profile)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// deleteUserProfile deletes a single user profile and waits for deletion.
func deleteUserProfile(ctx context.Context, client SageMakerStudioAPI, domainID string, profile types.UserProfileDetails) error {
	if _, err := client.DeleteUserProfile(ctx, &sagemaker.DeleteUserProfileInput{
		DomainId:        aws.String(domainID),
		UserProfileName: profile.UserProfileName,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	return waitForDeletion(ctx, client, fmt.Sprintf("user profile %s", aws.ToString(profile.UserProfileName)), func() (bool, error) {
		profiles, err := client.ListUserProfiles(ctx, &sagemaker.ListUserProfilesInput{DomainIdEquals: aws.String(domainID)})
		if err != nil {
			return false, err
		}
		for _, p := range profiles.UserProfiles {
			if aws.ToString(p.UserProfileName) == aws.ToString(profile.UserProfileName) {
				return true, nil
			}
		}
		return false, nil
	})
}

// waitForDeletion polls until a resource is confirmed deleted or timeout occurs.
func waitForDeletion(ctx context.Context, client SageMakerStudioAPI, resourceName string, checkExists func() (bool, error)) error {
	startTime := time.Now()
	for {
		if time.Since(startTime) > sageMakerMaxWaitTime {
			return fmt.Errorf("timeout waiting for %s deletion after %v", resourceName, sageMakerMaxWaitTime)
		}

		exists, err := checkExists()
		if err != nil {
			return err
		}
		if !exists {
			logging.Debugf("[OK] %s deletion confirmed", resourceName)
			return nil
		}

		time.Sleep(sageMakerRetryDelay)
	}
}
