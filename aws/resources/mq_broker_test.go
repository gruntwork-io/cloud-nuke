package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockMQBrokerClient struct {
	ListBrokersOutput  mq.ListBrokersOutput
	TagsByARN          map[string]map[string]string
	DeleteBrokerOutput mq.DeleteBrokerOutput
}

func (m *mockMQBrokerClient) ListBrokers(ctx context.Context, params *mq.ListBrokersInput, optFns ...func(*mq.Options)) (*mq.ListBrokersOutput, error) {
	return &m.ListBrokersOutput, nil
}

func (m *mockMQBrokerClient) ListTags(ctx context.Context, params *mq.ListTagsInput, optFns ...func(*mq.Options)) (*mq.ListTagsOutput, error) {
	if m.TagsByARN != nil {
		if tags, ok := m.TagsByARN[aws.ToString(params.ResourceArn)]; ok {
			return &mq.ListTagsOutput{Tags: tags}, nil
		}
	}
	return &mq.ListTagsOutput{}, nil
}

func (m *mockMQBrokerClient) DeleteBroker(ctx context.Context, params *mq.DeleteBrokerInput, optFns ...func(*mq.Options)) (*mq.DeleteBrokerOutput, error) {
	return &m.DeleteBrokerOutput, nil
}

func TestListMQBrokers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockMQBrokerClient{
		ListBrokersOutput: mq.ListBrokersOutput{
			BrokerSummaries: []types.BrokerSummary{
				{BrokerName: aws.String("broker1"), BrokerId: aws.String("b-1111"), BrokerArn: aws.String("arn::broker1"), Created: aws.Time(now), BrokerState: types.BrokerStateRunning, DeploymentMode: types.DeploymentModeSingleInstance, EngineType: types.EngineTypeActivemq},
				{BrokerName: aws.String("broker2"), BrokerId: aws.String("b-2222"), BrokerArn: aws.String("arn::broker2"), Created: aws.Time(now.Add(1 * time.Hour)), BrokerState: types.BrokerStateRunning, DeploymentMode: types.DeploymentModeSingleInstance, EngineType: types.EngineTypeRabbitmq},
			},
		},
		TagsByARN: map[string]map[string]string{
			"arn::broker1": {"env": "prod"},
			"arn::broker2": {"env": "dev"},
		},
	}

	tests := map[string]struct {
		mock      *mockMQBrokerClient
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"b-1111", "b-2222"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("broker1")}},
				},
			},
			expected: []string{"b-2222"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{"b-1111"},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^prod$")}},
				},
			},
			expected: []string{"b-1111"},
		},
		"skipNonDeletableBrokers": {
			mock: &mockMQBrokerClient{
				ListBrokersOutput: mq.ListBrokersOutput{
					BrokerSummaries: []types.BrokerSummary{
						{BrokerName: aws.String("broker1"), BrokerId: aws.String("b-1111"), BrokerArn: aws.String("arn::broker1"), Created: aws.Time(now), BrokerState: types.BrokerStateRunning, DeploymentMode: types.DeploymentModeSingleInstance, EngineType: types.EngineTypeActivemq},
						{BrokerName: aws.String("broker2"), BrokerId: aws.String("b-2222"), BrokerArn: aws.String("arn::broker2"), Created: aws.Time(now), BrokerState: types.BrokerStateDeletionInProgress, DeploymentMode: types.DeploymentModeSingleInstance, EngineType: types.EngineTypeActivemq},
						{BrokerName: aws.String("broker3"), BrokerId: aws.String("b-3333"), BrokerArn: aws.String("arn::broker3"), Created: aws.Time(now), BrokerState: types.BrokerStateCreationInProgress, DeploymentMode: types.DeploymentModeSingleInstance, EngineType: types.EngineTypeActivemq},
						{BrokerName: aws.String("broker4"), BrokerId: aws.String("b-4444"), BrokerArn: aws.String("arn::broker4"), Created: aws.Time(now), BrokerState: types.BrokerStateCreationFailed, DeploymentMode: types.DeploymentModeSingleInstance, EngineType: types.EngineTypeActivemq},
					},
				},
			},
			configObj: config.ResourceType{},
			expected:  []string{"b-1111"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mock
			if tc.mock != nil {
				client = tc.mock
			}
			ids, err := listMQBrokers(context.Background(), client, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteMQBroker(t *testing.T) {
	t.Parallel()
	err := deleteMQBroker(context.Background(), &mockMQBrokerClient{}, aws.String("b-1111"))
	require.NoError(t, err)
}
