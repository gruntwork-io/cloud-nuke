package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
)

// SageMakerEndpointConfigAPI represents the shape of the AWS SageMaker API for testing
// This interface allows for mocking in tests and defines all required SageMaker operations
type SageMakerEndpointConfigAPI interface {
	ListEndpointConfigs(ctx context.Context, params *sagemaker.ListEndpointConfigsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointConfigsOutput, error)
	DeleteEndpointConfig(ctx context.Context, params *sagemaker.DeleteEndpointConfigInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointConfigOutput, error)
	ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error)
}

// SageMakerEndpointConfig represents an AWS SageMaker Endpoint Configuration resource to be nuked
// It implements the Resource interface and handles the deletion of SageMaker Endpoint Configurations
type SageMakerEndpointConfig struct {
	BaseAwsResource
	Client              SageMakerEndpointConfigAPI // AWS SageMaker client or mock for testing
	Region              string                     // AWS region where the resources are located
	EndpointConfigNames []string                   // List of SageMaker Endpoint Configuration names to be nuked
}

// Init initializes the SageMaker client with the provided AWS configuration
func (s *SageMakerEndpointConfig) Init(cfg aws.Config) {
	s.Client = sagemaker.NewFromConfig(cfg)
}

// ResourceName returns the identifier string for this resource type
// Used for logging and reporting purposes
func (s *SageMakerEndpointConfig) ResourceName() string {
	return "sagemaker-endpoint-config"
}

// ResourceIdentifiers returns the list of endpoint configuration names that will be nuked
func (s *SageMakerEndpointConfig) ResourceIdentifiers() []string {
	return s.EndpointConfigNames
}

// MaxBatchSize returns the maximum number of resources that can be deleted in parallel
// This limit helps prevent AWS API throttling
func (s *SageMakerEndpointConfig) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}

// GetAndSetResourceConfig retrieves the configuration for SageMaker Endpoint Configuration resources
func (s *SageMakerEndpointConfig) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SageMakerEndpointConfig
}

// GetAndSetIdentifiers retrieves all SageMaker Endpoint Configurations and stores their identifiers
func (s *SageMakerEndpointConfig) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := s.GetAll(c, configObj)
	if err != nil {
		return nil, err
	}

	s.EndpointConfigNames = aws.ToStringSlice(identifiers)
	return s.EndpointConfigNames, nil
}

// IsNukable determines whether a specific SageMaker Endpoint Configuration can be deleted
// It will respect tag exclusion rules from the configuration
func (s *SageMakerEndpointConfig) IsNukable(identifier string) (bool, error) {
	// Look for endpoint config with this name in our list
	for _, endpointConfigName := range s.EndpointConfigNames {
		if endpointConfigName == identifier {
			// This endpoint config was already vetted through our filtering process
			// and is included in the list of endpoint configs to nuke
			return true, nil
		}
	}

	// If we can't find it in our approved list, don't allow it to be nuked
	// This means either it was filtered out by tag exclusion rules or
	// it doesn't exist in the current account/region
	return false, fmt.Errorf("endpoint configuration is protected by tag exclusion rules or not found")
}

// Nuke implements the AwsResource interface by calling nukeAll
// It takes a list of endpoint configuration identifiers and deletes those endpoint configs
func (s *SageMakerEndpointConfig) Nuke(identifiers []string) error {
	return s.nukeAll(identifiers)
}
