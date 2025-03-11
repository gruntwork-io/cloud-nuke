package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// SageMakerStudioAPI represents the shape of the AWS SageMaker Studio API for testing
// This interface allows for mocking in tests and defines all required SageMaker operations
type SageMakerStudioAPI interface {
	// Domain operations
	ListDomains(ctx context.Context, params *sagemaker.ListDomainsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListDomainsOutput, error)
	DescribeDomain(ctx context.Context, params *sagemaker.DescribeDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeDomainOutput, error)
	DeleteDomain(ctx context.Context, params *sagemaker.DeleteDomainInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteDomainOutput, error)

	// UserProfile operations
	ListUserProfiles(ctx context.Context, params *sagemaker.ListUserProfilesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListUserProfilesOutput, error)
	DescribeUserProfile(ctx context.Context, params *sagemaker.DescribeUserProfileInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeUserProfileOutput, error)
	DeleteUserProfile(ctx context.Context, params *sagemaker.DeleteUserProfileInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteUserProfileOutput, error)

	// App operations
	ListApps(ctx context.Context, params *sagemaker.ListAppsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListAppsOutput, error)
	DescribeApp(ctx context.Context, params *sagemaker.DescribeAppInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeAppOutput, error)
	DeleteApp(ctx context.Context, params *sagemaker.DeleteAppInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteAppOutput, error)

	// Space operations
	ListSpaces(ctx context.Context, params *sagemaker.ListSpacesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListSpacesOutput, error)
	DeleteSpace(ctx context.Context, params *sagemaker.DeleteSpaceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteSpaceOutput, error)

	// MLflow tracking server operations
	ListMlflowTrackingServers(ctx context.Context, params *sagemaker.ListMlflowTrackingServersInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListMlflowTrackingServersOutput, error)
	DeleteMlflowTrackingServer(ctx context.Context, params *sagemaker.DeleteMlflowTrackingServerInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteMlflowTrackingServerOutput, error)
	DescribeMlflowTrackingServer(ctx context.Context, params *sagemaker.DescribeMlflowTrackingServerInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeMlflowTrackingServerOutput, error)
}

// SageMakerStudio represents an AWS SageMaker Studio resource to be nuked
// It implements the Resource interface and handles the deletion of SageMaker Studio
// resources including domains, user profiles, apps, spaces, and MLflow tracking servers
type SageMakerStudio struct {
	BaseAwsResource
	Client            SageMakerStudioAPI // AWS SageMaker client or mock for testing
	Region            string             // AWS region where the resources are located
	StudioDomainNames []string           // List of SageMaker Studio domain names to be nuked
}

// Init initializes the SageMaker client with the provided AWS configuration
// This method must be called before using any other methods
func (s *SageMakerStudio) Init(cfg aws.Config) {
	s.Client = sagemaker.NewFromConfig(cfg)
	s.BaseAwsResource.Init(cfg)
}

// ResourceName returns the identifier string for this resource type
// Used for logging and reporting purposes
func (s *SageMakerStudio) ResourceName() string {
	return "sagemaker-studio"
}

// ResourceIdentifiers returns the list of domain names that will be nuked
// These identifiers are used to track and report the deletion progress
func (s *SageMakerStudio) ResourceIdentifiers() []string {
	return s.StudioDomainNames
}

// MaxBatchSize returns the maximum number of resources that can be deleted in parallel
// This limit helps prevent AWS API throttling
func (s *SageMakerStudio) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// GetAndSetResourceConfig retrieves the configuration for SageMaker Studio resources
// from the provided config object and converts it to a ResourceType
func (s *SageMakerStudio) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	// Convert SageMakerStudioDomainResourceType to ResourceType
	return config.ResourceType{
		Timeout:            configObj.SageMakerStudioDomain.Timeout,
		ProtectUntilExpire: configObj.SageMakerStudioDomain.ProtectUntilExpire,
	}
}

// GetAndSetIdentifiers retrieves all SageMaker Studio domains and stores their identifiers
// It applies any filtering rules from the configuration
func (s *SageMakerStudio) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := s.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	s.StudioDomainNames = aws.ToStringSlice(identifiers)
	return s.StudioDomainNames, nil
}

// IsNukable determines whether a specific SageMaker Studio domain can be deleted
// Currently always returns true as all domains are considered nukable
func (s *SageMakerStudio) IsNukable(identifier string) (bool, error) {
	return true, nil
}

// Nuke deletes the specified SageMaker Studio domains and all their associated resources
// This includes user profiles, apps, spaces, and MLflow tracking servers
func (s *SageMakerStudio) Nuke(identifiers []string) error {
	if err := s.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
