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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockCloudMapServicesClient struct {
	ListServicesOutput        servicediscovery.ListServicesOutput
	DeleteServiceOutput       servicediscovery.DeleteServiceOutput
	ListInstancesOutput       servicediscovery.ListInstancesOutput
	DeregisterInstanceOutput  servicediscovery.DeregisterInstanceOutput
	ListTagsForResourceOutput servicediscovery.ListTagsForResourceOutput
	ListTagsForResourceFn     func(arn string) *servicediscovery.ListTagsForResourceOutput
	ListInstancesFn           func() *servicediscovery.ListInstancesOutput
}

func (m *mockCloudMapServicesClient) ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m *mockCloudMapServicesClient) DeleteService(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m *mockCloudMapServicesClient) ListInstances(ctx context.Context, params *servicediscovery.ListInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListInstancesOutput, error) {
	if m.ListInstancesFn != nil {
		return m.ListInstancesFn(), nil
	}
	return &m.ListInstancesOutput, nil
}

func (m *mockCloudMapServicesClient) DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
	return &m.DeregisterInstanceOutput, nil
}

func (m *mockCloudMapServicesClient) ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
	if m.ListTagsForResourceFn != nil && params.ResourceARN != nil {
		return m.ListTagsForResourceFn(*params.ResourceARN), nil
	}
	return &m.ListTagsForResourceOutput, nil
}

func TestListCloudMapServices(t *testing.T) {
	t.Parallel()

	testId1 := "srv-123456789"
	testId2 := "srv-987654321"
	testName1 := "test-service-1"
	testName2 := "test-service-2"
	testArn1 := "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-123456789"
	testArn2 := "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-987654321"
	now := time.Now()

	mock := &mockCloudMapServicesClient{
		ListServicesOutput: servicediscovery.ListServicesOutput{
			Services: []types.ServiceSummary{
				{
					Id:         aws.String(testId1),
					Arn:        aws.String(testArn1),
					Name:       aws.String(testName1),
					CreateDate: aws.Time(now),
				},
				{
					Id:         aws.String(testId2),
					Arn:        aws.String(testArn2),
					Name:       aws.String(testName2),
					CreateDate: aws.Time(now.Add(1 * time.Hour)),
				},
			},
		},
		ListTagsForResourceFn: func(arn string) *servicediscovery.ListTagsForResourceOutput {
			if arn == testArn1 {
				return &servicediscovery.ListTagsForResourceOutput{
					Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
				}
			}
			return &servicediscovery.ListTagsForResourceOutput{
				Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
			}
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testId1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{testId2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listCloudMapServices(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteCloudMapService(t *testing.T) {
	t.Parallel()

	mock := &mockCloudMapServicesClient{
		ListInstancesOutput: servicediscovery.ListInstancesOutput{
			Instances: []types.InstanceSummary{}, // No instances
		},
		DeleteServiceOutput: servicediscovery.DeleteServiceOutput{},
	}

	err := deleteCloudMapService(context.Background(), mock, aws.String("srv-123456789"))
	require.NoError(t, err)
}

func TestDeleteCloudMapServiceWithInstances(t *testing.T) {
	t.Parallel()

	listCallCount := 0
	mock := &mockCloudMapServicesClient{
		DeregisterInstanceOutput: servicediscovery.DeregisterInstanceOutput{
			OperationId: aws.String("op-123"),
		},
		DeleteServiceOutput: servicediscovery.DeleteServiceOutput{},
		ListInstancesFn: func() *servicediscovery.ListInstancesOutput {
			listCallCount++
			if listCallCount == 1 {
				// First call during deregistration - return instances
				return &servicediscovery.ListInstancesOutput{
					Instances: []types.InstanceSummary{
						{Id: aws.String("instance-1")},
					},
				}
			}
			// Subsequent calls during wait - return empty (instances deregistered)
			return &servicediscovery.ListInstancesOutput{
				Instances: []types.InstanceSummary{},
			}
		},
	}

	err := deleteCloudMapService(context.Background(), mock, aws.String("srv-123456789"))
	require.NoError(t, err)
}
