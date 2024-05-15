package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEC2Service struct {
	ecsiface.ECSAPI
	ListClustersOutput     ecs.ListClustersOutput
	DescribeServicesOutput ecs.DescribeServicesOutput
	ListServicesOutput     ecs.ListServicesOutput
	UpdateServiceOutput    ecs.UpdateServiceOutput
	DeleteServiceOutput    ecs.DeleteServiceOutput
}

func (m mockedEC2Service) ListClustersWithContext(_ aws.Context, _ *ecs.ListClustersInput, _ ...request.Option) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedEC2Service) DescribeServicesWithContext(_ aws.Context, _ *ecs.DescribeServicesInput, _ ...request.Option) (*ecs.DescribeServicesOutput, error) {
	return &m.DescribeServicesOutput, nil
}

func (m mockedEC2Service) ListServicesWithContext(_ aws.Context, _ *ecs.ListServicesInput, _ ...request.Option) (*ecs.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func (m mockedEC2Service) UpdateServiceWithContext(_ aws.Context, _ *ecs.UpdateServiceInput, _ ...request.Option) (*ecs.UpdateServiceOutput, error) {
	return &m.UpdateServiceOutput, nil
}

func (m mockedEC2Service) WaitUntilServicesStableWithContext(_ aws.Context, _ *ecs.DescribeServicesInput, _ ...request.WaiterOption) error {
	return nil
}

func (m mockedEC2Service) DeleteServiceWithContext(_ aws.Context, _ *ecs.DeleteServiceInput, _ ...request.Option) (*ecs.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m mockedEC2Service) WaitUntilServicesInactiveWithContext(_ aws.Context, _ *ecs.DescribeServicesInput, _ ...request.WaiterOption) error {
	return nil
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
				ClusterArns: []*string{
					aws.String(testArn1),
				},
			},
			ListServicesOutput: ecs.ListServicesOutput{
				ServiceArns: []*string{
					aws.String(testArn1),
				},
			},
			DescribeServicesOutput: ecs.DescribeServicesOutput{
				Services: []*ecs.Service{
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestEC2Service_NukeAll(t *testing.T) {

	t.Parallel()

	es := ECSServices{
		Client: mockedEC2Service{
			DescribeServicesOutput: ecs.DescribeServicesOutput{
				Services: []*ecs.Service{
					{
						ServiceArn:         aws.String("testArn1"),
						SchedulingStrategy: aws.String(ecs.SchedulingStrategyDaemon),
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
