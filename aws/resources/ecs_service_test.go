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
}

func (m mockedECSService) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedECSService) ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m mockedECSService) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return &m.DescribeServicesOutput, nil
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
	mockClient := mockedECSService{
		ListClustersOutput: ecs.ListClustersOutput{
			ClusterArns: []string{
				testArn1,
			},
		},
		ListServicesOutput: ecs.ListServicesOutput{
			ServiceArns: []string{
				testArn1,
			},
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
			names, err := listECSServices(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestECSService_NukeAll(t *testing.T) {
	t.Parallel()

	// Setup global state with service cluster map
	globalECSServicesState.serviceClusterMap = map[string]string{
		"testArn1": "testCluster1",
	}

	mockClient := mockedECSService{
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

	err := deleteECSServices(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, "ecsserv", []*string{aws.String("testArn1")})
	require.NoError(t, err)
}
