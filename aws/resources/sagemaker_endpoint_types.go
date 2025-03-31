package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
)

// SageMakerAPI represents the shape of the AWS SageMaker API for testing
// This interface allows for mocking in tests and defines all required SageMaker operations
type SageMakerAPI interface {
	ListEndpoints(ctx context.Context, params *sagemaker.ListEndpointsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointsOutput, error)
	DeleteEndpoint(ctx context.Context, params *sagemaker.DeleteEndpointInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointOutput, error)
	ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error)
}

// SageMakerEndpoint represents an AWS SageMaker Endpoint resource to be nuked
// It implements the Resource interface and handles the deletion of SageMaker Endpoints
type SageMakerEndpoint struct {
	BaseAwsResource
	Client        SageMakerAPI // AWS SageMaker client or mock for testing
	Region        string       // AWS region where the resources are located
	EndpointNames []string     // List of SageMaker Endpoint names to be nuked
}

// Init initializes the SageMaker client with the provided AWS configuration
func (s *SageMakerEndpoint) Init(cfg aws.Config) {
	s.Client = sagemaker.NewFromConfig(cfg)
}

// ResourceName returns the identifier string for this resource type
// Used for logging and reporting purposes
func (s *SageMakerEndpoint) ResourceName() string {
	return "sagemaker-endpoint"
}

// ResourceIdentifiers returns the list of endpoint names that will be nuked
func (s *SageMakerEndpoint) ResourceIdentifiers() []string {
	return s.EndpointNames
}

// MaxBatchSize returns the maximum number of resources that can be deleted in parallel
// This limit helps prevent AWS API throttling
func (s *SageMakerEndpoint) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

// GetAndSetResourceConfig retrieves the configuration for SageMaker Endpoint resources
func (s *SageMakerEndpoint) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SageMakerEndpoint
}

// GetAndSetIdentifiers retrieves all SageMaker Endpoints and stores their identifiers
func (s *SageMakerEndpoint) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := s.GetAll(c, configObj)
	if err != nil {
		return nil, err
	}

	s.EndpointNames = aws.ToStringSlice(identifiers)
	return s.EndpointNames, nil
}

// IsNukable determines whether a specific SageMaker Endpoint can be deleted
// It will respect tag exclusion rules from the configuration
func (s *SageMakerEndpoint) IsNukable(identifier string) (bool, error) {
	// Look for endpoint with this name in our list
	for _, endpointName := range s.EndpointNames {
		if endpointName == identifier {
			// This endpoint was already vetted through our filtering process
			// and is included in the list of endpoints to nuke
			return true, nil
		}
	}

	// If we can't find it in our approved list, don't allow it to be nuked
	// This means either it was filtered out by tag exclusion rules or
	// it doesn't exist in the current account/region
	return false, fmt.Errorf("endpoint is protected by tag exclusion rules or not found")
}

// Nuke implements the AwsResource interface by calling nukeAll
// It takes a list of endpoint identifiers and deletes those endpoints
func (s *SageMakerEndpoint) Nuke(identifiers []string) error {
	return s.nukeAll(identifiers)
}
