package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

// mockedCloudMapServicesAPI is a mock implementation of CloudMapServicesAPI for testing.
// It returns predefined responses for API calls, allowing tests to run without AWS credentials.
type mockedCloudMapServicesAPI struct {
	CloudMapServicesAPI
	ListServicesOutput        servicediscovery.ListServicesOutput
	DeleteServiceOutput       servicediscovery.DeleteServiceOutput
	ListInstancesOutput       servicediscovery.ListInstancesOutput
	DeregisterInstanceOutput  servicediscovery.DeregisterInstanceOutput
	ListTagsForResourceOutput servicediscovery.ListTagsForResourceOutput
}

func (m mockedCloudMapServicesAPI) ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m mockedCloudMapServicesAPI) DeleteService(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m mockedCloudMapServicesAPI) ListInstances(ctx context.Context, params *servicediscovery.ListInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListInstancesOutput, error) {
	return &m.ListInstancesOutput, nil
}

func (m mockedCloudMapServicesAPI) DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
	return &m.DeregisterInstanceOutput, nil
}

func (m mockedCloudMapServicesAPI) ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
	// Return different tags based on the service ARN
	if params.ResourceARN != nil {
		switch *params.ResourceARN {
		case "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-123456789":
			return &servicediscovery.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				},
			}, nil
		case "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-987654321":
			return &servicediscovery.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("production"),
					},
				},
			}, nil
		}
	}
	return &m.ListTagsForResourceOutput, nil
}

// TestCloudMapServices_GetAll verifies that service filtering works correctly.
// It tests empty filters, name exclusion, and time-based exclusion.
func TestCloudMapServices_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testService1 := "srv-123456789"
	testService2 := "srv-987654321"
	testService1Name := "test-service-1"
	testService2Name := "test-service-2"
	testService1Arn := "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-123456789"
	testService2Arn := "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-987654321"

	cms := CloudMapServices{
		Client: mockedCloudMapServicesAPI{
			ListServicesOutput: servicediscovery.ListServicesOutput{
				Services: []types.ServiceSummary{
					{
						Id:         aws.String(testService1),
						Arn:        aws.String(testService1Arn),
						Name:       aws.String(testService1Name),
						CreateDate: aws.Time(now.Add(-1 * time.Hour)),
					},
					{
						Id:         aws.String(testService2),
						Arn:        aws.String(testService2Arn),
						Name:       aws.String(testService2Name),
						CreateDate: aws.Time(now),
					},
				},
			},
			ListTagsForResourceOutput: servicediscovery.ListTagsForResourceOutput{
				Tags: []types.Tag{},
			},
		},
	}
	cms.BaseAwsResource.Init(aws.Config{})

	// Define test cases for different filter scenarios
	tests := map[string]struct {
		configObj config.Config
		expected  []string
	}{
		"emptyFilter": { // Should return all services when no filters are applied
			configObj: config.Config{
				CloudMapService: config.ResourceType{},
			},
			expected: []string{testService1, testService2},
		},
		"nameExclusionFilter": { // Should exclude services matching the regex pattern
			configObj: config.Config{
				CloudMapService: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile("test-service-1"),
						}},
					},
				},
			},
			expected: []string{testService2},
		},
		"timeAfterExclusionFilter": { // Should exclude services created after the specified time
			configObj: config.Config{
				CloudMapService: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now.Add(-30 * time.Minute)),
					},
				},
			},
			expected: []string{testService1},
		},
		"tagExclusionFilter": { // Should exclude services with tags matching the filter
			configObj: config.Config{
				CloudMapService: config.ResourceType{
					ExcludeRule: config.FilterRule{
						Tags: map[string]config.Expression{
							"Environment": {RE: *regexp.MustCompile("test")},
						},
					},
				},
			},
			expected: []string{testService2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := cms.getAll(context.Background(), tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

// TestCloudMapServices_NukeAll verifies that the service deletion process works correctly.
// It tests that services can be deleted after their instances are deregistered.
func TestCloudMapServices_NukeAll(t *testing.T) {
	t.Parallel()

	cms := CloudMapServices{
		Client: mockedCloudMapServicesAPI{
			ListInstancesOutput: servicediscovery.ListInstancesOutput{
				Instances: []types.InstanceSummary{},
			},
			DeregisterInstanceOutput: servicediscovery.DeregisterInstanceOutput{
				OperationId: aws.String("operation-123"),
			},
			DeleteServiceOutput: servicediscovery.DeleteServiceOutput{},
		},
	}
	cms.BaseAwsResource.Init(aws.Config{})

	err := cms.nukeAll([]*string{aws.String("srv-123456789")})
	require.NoError(t, err)
}
