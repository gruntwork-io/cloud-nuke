package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedECSService struct {
	ECSServicesAPI
	ListClustersOutput     ecs.ListClustersOutput
	ListServicesOutput     ecs.ListServicesOutput
	DescribeServicesOutput ecs.DescribeServicesOutput
	DeleteServiceOutput    ecs.DeleteServiceOutput
	UpdateServiceOutput    ecs.UpdateServiceOutput

	// Track calls to simulate service becoming inactive after delete
	describeServicesCalls int
}

func (m mockedECSService) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedECSService) ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m *mockedECSService) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	m.describeServicesCalls++
	// First call returns services (for scheduling strategy check)
	// Subsequent calls return INACTIVE status (for waiter to succeed)
	if m.describeServicesCalls <= 1 {
		return &m.DescribeServicesOutput, nil
	}
	// Return service with INACTIVE status so the waiter is satisfied
	return &ecs.DescribeServicesOutput{
		Services: []types.Service{
			{
				ServiceArn: aws.String("testArn1"),
				Status:     aws.String("INACTIVE"),
			},
		},
	}, nil
}

func (m mockedECSService) DeleteService(ctx context.Context, params *ecs.DeleteServiceInput, optFns ...func(*ecs.Options)) (*ecs.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m mockedECSService) UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
	return &m.UpdateServiceOutput, nil
}

func TestECSService_GetAll(t *testing.T) {
	t.Parallel()
	testArn1 := "testArn1"
	testArn2 := "testArn2"
	testName1 := "testService1"
	testName2 := "testService2"
	now := time.Now()

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testArn1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := &mockedECSService{
				ListClustersOutput: ecs.ListClustersOutput{
					ClusterArns: []string{testArn1},
				},
				ListServicesOutput: ecs.ListServicesOutput{
					ServiceArns: []string{testArn1},
				},
				DescribeServicesOutput: ecs.DescribeServicesOutput{
					Services: []types.Service{
						{
							ServiceArn:  aws.String(testArn1),
							ServiceName: aws.String(testName1),
							CreatedAt:   aws.Time(now),
						},
						{
							ServiceArn:  aws.String(testArn2),
							ServiceName: aws.String(testName2),
							CreatedAt:   aws.Time(now.Add(1)),
						},
					},
				},
			}
			serviceClusterMap := make(map[string]string)
			names, err := listECSServices(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, tc.configObj, serviceClusterMap)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestECSService_NukeAll(t *testing.T) {
	t.Parallel()

	// Service cluster map holds the service-to-cluster mapping needed for deletion
	serviceClusterMap := map[string]string{
		"testArn1": "testCluster1",
	}

	mockClient := &mockedECSService{
		DescribeServicesOutput: ecs.DescribeServicesOutput{
			Services: []types.Service{
				{
					SchedulingStrategy: types.SchedulingStrategyDaemon,
					ServiceArn:         aws.String("testArn1"),
					Status:             aws.String("DRAINING"),
				},
			},
		},
		UpdateServiceOutput: ecs.UpdateServiceOutput{},
		DeleteServiceOutput: ecs.DeleteServiceOutput{},
	}

	results := deleteECSServices(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, "ecsserv", []*string{aws.String("testArn1")}, serviceClusterMap)
	require.Len(t, results, 1)
	require.Equal(t, "testArn1", results[0].Identifier)
}
