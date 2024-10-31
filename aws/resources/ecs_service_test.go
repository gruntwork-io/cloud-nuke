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
	"github.com/stretchr/testify/require"
)

type mockedEC2Service struct {
	ECSServicesAPI
	ListClustersOutput     ecs.ListClustersOutput
	ListServicesOutput     ecs.ListServicesOutput
	DescribeServicesOutput ecs.DescribeServicesOutput
	DeleteServiceOutput    ecs.DeleteServiceOutput
	UpdateServiceOutput    ecs.UpdateServiceOutput
}

func (m mockedEC2Service) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedEC2Service) ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m mockedEC2Service) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return &m.DescribeServicesOutput, nil
}

func (m mockedEC2Service) DeleteService(ctx context.Context, params *ecs.DeleteServiceInput, optFns ...func(*ecs.Options)) (*ecs.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m mockedEC2Service) UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error) {
	return &m.UpdateServiceOutput, nil
}

func TestEC2Service_GetAll(t *testing.T) {
	t.Parallel()
	testArn1 := "testArn1"
	testArn2 := "testArn2"
	testName1 := "testService1"
	testName2 := "testService2"
	now := time.Now()
	es := ECSServices{
		Client: mockedEC2Service{
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
			names, err := es.getAll(context.Background(), config.Config{
				ECSService: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestEC2Service_NukeAll(t *testing.T) {
	t.Parallel()
	es := ECSServices{
		BaseAwsResource: BaseAwsResource{
			Context: context.Background(),
		},
		Client: mockedEC2Service{
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
		},
	}

	err := es.nukeAll([]*string{aws.String("testArn1")})
	require.NoError(t, err)
}
