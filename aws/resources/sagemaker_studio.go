package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	goerrors "github.com/gruntwork-io/go-commons/errors"
	"golang.org/x/sync/errgroup"
)

const (
	// sagemakerRetryInitialDelay is the initial delay between retries for SageMaker operations
	sagemakerRetryInitialDelay = 10 * time.Second
	// sagemakerRetryMaxDelay is the maximum delay between retries for SageMaker operations
	sagemakerRetryMaxDelay = 2 * time.Minute
	// sagemakerMaxRetries is the maximum number of retry attempts for SageMaker operations
	sagemakerMaxRetries = 3
	// maxWaitTime is the maximum time to wait for any resource deletion
	maxWaitTime = 30 * time.Minute
	// retryDelay is the delay between retry attempts
	retryDelay = 10 * time.Second
)

// waitForResourceDeletion is a generic wait function that polls until a resource is confirmed to be deleted
// It uses exponential backoff with a maximum wait time
type resourceChecker func() (bool, error)

// deleteResourceWithRetry is a helper function that handles resource deletion with retry logic
func (s *SageMakerStudio) deleteResourceWithRetry(
	resourceType string,
	identifier string,
	deleteFunc func() error,
	waitFunc func() error,
) error {
	delay := sagemakerRetryInitialDelay
	var lastErr error

	for i := 0; i < sagemakerMaxRetries; i++ {
		logging.Debugf("Attempting to delete %s %s (attempt %d/%d)", resourceType, identifier, i+1, sagemakerMaxRetries)

		if err := deleteFunc(); err != nil {
			lastErr = err
			logging.Debugf("[Retrying] Delete %s %s in %s: %s", resourceType, identifier, s.Region, err)
			time.Sleep(delay)
			if delay < sagemakerRetryMaxDelay {
				delay *= 2
			}
			continue
		}

		if err := waitFunc(); err != nil {
			lastErr = err
			continue
		}

		report.Record(report.Entry{
			Identifier:   identifier,
			ResourceType: resourceType,
			Error:        nil,
		})
		return nil
	}

	// Record failed deletion
	report.Record(report.Entry{
		Identifier:   identifier,
		ResourceType: resourceType,
		Error:        lastErr,
	})
	return goerrors.WithStackTrace(fmt.Errorf("failed to delete %s %s after %d retries: %v", resourceType, identifier, sagemakerMaxRetries, lastErr))
}

// listUserProfiles lists all user profiles for a domain
func (s *SageMakerStudio) listUserProfiles(domainID string) ([]types.UserProfileDetails, error) {
	var profiles []types.UserProfileDetails
	paginator := sagemaker.NewListUserProfilesPaginator(s.Client, &sagemaker.ListUserProfilesInput{
		DomainIdEquals: aws.String(domainID),
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(s.Context)
		if err != nil {
			logging.Debugf("[Failed] Error listing user profiles in %s: %s", s.Region, err)
			return nil, goerrors.WithStackTrace(err)
		}
		profiles = append(profiles, output.UserProfiles...)
	}
	return profiles, nil
}

// listSpaces lists all spaces for a domain
func (s *SageMakerStudio) listSpaces(domainID string) ([]types.SpaceDetails, error) {
	var spaces []types.SpaceDetails
	paginator := sagemaker.NewListSpacesPaginator(s.Client, &sagemaker.ListSpacesInput{
		DomainIdEquals: aws.String(domainID),
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(s.Context)
		if err != nil {
			logging.Debugf("[Failed] Error listing spaces in %s: %s", s.Region, err)
			return nil, goerrors.WithStackTrace(err)
		}
		spaces = append(spaces, output.Spaces...)
	}
	return spaces, nil
}

// listAppsForResource lists all apps for a given resource (user profile or space)
func (s *SageMakerStudio) listAppsForResource(domainID string, resourceName *string, isUserProfile bool) ([]types.AppDetails, error) {
	input := &sagemaker.ListAppsInput{
		DomainIdEquals: aws.String(domainID),
	}
	if isUserProfile {
		input.UserProfileNameEquals = resourceName
	} else {
		input.SpaceNameEquals = resourceName
	}

	apps, err := s.Client.ListApps(s.Context, input)
	if err != nil {
		resourceType := "space"
		if isUserProfile {
			resourceType = "user profile"
		}
		return nil, goerrors.WithStackTrace(
			fmt.Errorf("error listing apps for %s %s in %s: %v",
				resourceType, *resourceName, s.Region, err))
	}
	return apps.Apps, nil
}

// waitForDeletion polls until a resource is confirmed deleted or timeout occurs
func (s *SageMakerStudio) waitForDeletion(resourceName string, maxWait time.Duration, checkExists func() (bool, error)) error {
	startTime := time.Now()
	for {
		if time.Since(startTime) > maxWait {
			return fmt.Errorf("timeout waiting for %s deletion after %v", resourceName, maxWait)
		}

		exists, err := checkExists()
		if err != nil {
			return err
		}
		if !exists {
			logging.Debugf("[OK] %s deletion confirmed in %s", resourceName, s.Region)
			return nil
		}

		time.Sleep(retryDelay)
	}
}

// nukeDomain deletes a SageMaker Studio domain and all its resources
func (s *SageMakerStudio) nukeDomain(domainID string) error {
	logging.Debugf("Starting deletion of SageMaker Studio domain %s in region %s", domainID, s.Region)

	// Delete apps, spaces, and MLflow tracking servers in parallel
	g := new(errgroup.Group)

	// Delete apps
	if apps, err := s.Client.ListApps(s.Context, &sagemaker.ListAppsInput{DomainIdEquals: aws.String(domainID)}); err == nil {
		for _, app := range apps.Apps {
			if app.Status == types.AppStatusDeleted {
				continue
			}
			app := app
			g.Go(func() error {
				input := &sagemaker.DeleteAppInput{
					AppName: app.AppName, AppType: app.AppType, DomainId: aws.String(domainID),
					UserProfileName: app.UserProfileName, SpaceName: app.SpaceName,
				}
				_, err := s.Client.DeleteApp(s.Context, input)
				if err != nil {
					return err
				}

				return s.waitForDeletion(fmt.Sprintf("app %s", *app.AppName), maxWaitTime, func() (bool, error) {
					apps, err := s.Client.ListApps(s.Context, &sagemaker.ListAppsInput{DomainIdEquals: aws.String(domainID)})
					if err != nil {
						return false, err
					}
					for _, a := range apps.Apps {
						if *a.AppName == *app.AppName && a.Status != types.AppStatusDeleted {
							return true, nil
						}
					}
					return false, nil
				})
			})
		}
	}

	// Delete spaces
	if spaces, err := s.Client.ListSpaces(s.Context, &sagemaker.ListSpacesInput{DomainIdEquals: aws.String(domainID)}); err == nil {
		for _, space := range spaces.Spaces {
			space := space
			g.Go(func() error {
				_, err := s.Client.DeleteSpace(s.Context, &sagemaker.DeleteSpaceInput{
					DomainId: aws.String(domainID), SpaceName: space.SpaceName,
				})
				if err != nil {
					return err
				}

				return s.waitForDeletion(fmt.Sprintf("space %s", *space.SpaceName), maxWaitTime, func() (bool, error) {
					spaces, err := s.Client.ListSpaces(s.Context, &sagemaker.ListSpacesInput{DomainIdEquals: aws.String(domainID)})
					if err != nil {
						return false, err
					}
					for _, s := range spaces.Spaces {
						if *s.SpaceName == *space.SpaceName {
							return true, nil
						}
					}
					return false, nil
				})
			})
		}
	}

	// Delete MLflow tracking servers
	if servers, err := s.Client.ListMlflowTrackingServers(s.Context, &sagemaker.ListMlflowTrackingServersInput{}); err == nil {
		for _, server := range servers.TrackingServerSummaries {
			server := server
			g.Go(func() error {
				return s.deleteResourceWithRetry(
					"MLflow tracking server",
					*server.TrackingServerName,
					func() error {
						_, err := s.Client.DeleteMlflowTrackingServer(s.Context, &sagemaker.DeleteMlflowTrackingServerInput{
							TrackingServerName: server.TrackingServerName,
						})
						return err
					},
					func() error {
						return s.waitForDeletion(fmt.Sprintf("MLflow server %s", *server.TrackingServerName), maxWaitTime, func() (bool, error) {
							servers, err := s.Client.ListMlflowTrackingServers(s.Context, &sagemaker.ListMlflowTrackingServersInput{})
							if err != nil {
								return false, err
							}
							for _, s := range servers.TrackingServerSummaries {
								if *s.TrackingServerName == *server.TrackingServerName {
									return true, nil
								}
							}
							return false, nil
						})
					},
				)
			})
		}
	}

	// Wait for all app, spaces and tracking servers to be deleted
	if err := g.Wait(); err != nil {
		return err
	}

	// Delete user profiles
	if profiles, err := s.Client.ListUserProfiles(s.Context, &sagemaker.ListUserProfilesInput{DomainIdEquals: aws.String(domainID)}); err == nil {
		g = new(errgroup.Group)
		for _, profile := range profiles.UserProfiles {
			profile := profile
			g.Go(func() error {
				_, err := s.Client.DeleteUserProfile(s.Context, &sagemaker.DeleteUserProfileInput{
					DomainId: aws.String(domainID), UserProfileName: profile.UserProfileName,
				})
				if err != nil {
					return err
				}

				return s.waitForDeletion(fmt.Sprintf("user profile %s", *profile.UserProfileName), maxWaitTime, func() (bool, error) {
					profiles, err := s.Client.ListUserProfiles(s.Context, &sagemaker.ListUserProfilesInput{DomainIdEquals: aws.String(domainID)})
					if err != nil {
						return false, err
					}
					for _, p := range profiles.UserProfiles {
						if *p.UserProfileName == *profile.UserProfileName {
							return true, nil
						}
					}
					return false, nil
				})
			})
		}
		// Wait for all users to be deleted
		if err := g.Wait(); err != nil {
			return err
		}
	}

	// Delete the domain
	_, err := s.Client.DeleteDomain(s.Context, &sagemaker.DeleteDomainInput{
		DomainId:        aws.String(domainID),
		RetentionPolicy: &types.RetentionPolicy{HomeEfsFileSystem: types.RetentionTypeDelete},
	})
	if err != nil {
		return err
	}

	return s.waitForDeletion(fmt.Sprintf("domain %s", domainID), maxWaitTime, func() (bool, error) {
		_, err := s.Client.DescribeDomain(s.Context, &sagemaker.DescribeDomainInput{DomainId: aws.String(domainID)})
		return err == nil, nil
	})
}

// getAll retrieves all SageMaker Studio domains in the region
func (s *SageMakerStudio) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var domainIDs []*string
	paginator := sagemaker.NewListDomainsPaginator(s.Client, &sagemaker.ListDomainsInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(c)
		if err != nil {
			return nil, goerrors.WithStackTrace(err)
		}

		for _, domain := range output.Domains {
			if domain.DomainId != nil {
				logging.Debugf("Found SageMaker Studio domain: %s (Status: %s)", *domain.DomainId, domain.Status)
				domainIDs = append(domainIDs, domain.DomainId)
			}
		}
	}

	return domainIDs, nil
}

// nukeAll deletes all provided SageMaker Studio domains and their resources
func (s *SageMakerStudio) nukeAll(identifiers []string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No SageMaker Studio domains to nuke in region %s", s.Region)
		return nil
	}

	g := new(errgroup.Group)
	for _, domainID := range identifiers {
		if domainID == "" {
			continue
		}
		domainID := domainID
		g.Go(func() error {
			err := s.nukeDomain(domainID)
			report.Record(report.Entry{
				Identifier:   domainID,
				ResourceType: "SageMaker Studio Domain",
				Error:        err,
			})
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return goerrors.WithStackTrace(err)
	}

	logging.Debugf("[OK] %d SageMaker Studio domain(s) deleted in %s", len(identifiers), s.Region)
	return nil
}
