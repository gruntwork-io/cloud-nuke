package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedEventBridgeRuleService struct {
	EventBridgeRuleAPI
	ListEventBusesOutput    eventbridge.ListEventBusesOutput
	ListRulesOutput         eventbridge.ListRulesOutput
	ListTargetsByRuleOutput eventbridge.ListTargetsByRuleOutput

	DeleteRuleOutput eventbridge.DeleteRuleOutput
}

func (m mockedEventBridgeRuleService) ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error) {
	return &m.ListEventBusesOutput, nil
}

func (m mockedEventBridgeRuleService) ListRules(ctx context.Context, params *eventbridge.ListRulesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRulesOutput, error) {
	return &m.ListRulesOutput, nil
}

func (m mockedEventBridgeRuleService) ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error) {
	return &m.ListTargetsByRuleOutput, nil
}

func (m mockedEventBridgeRuleService) DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error) {
	return &m.DeleteRuleOutput, nil
}

func Test_EventBridgeRule_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()

	rule1 := "rule-1"
	rule2 := "rule-2"
	busName := "test-bus"
	bRule1 := fmt.Sprintf("%s|%s", busName, rule1)
	bRule2 := fmt.Sprintf("%s|%s", busName, rule2)

	service := EventBridgeRule{
		Client: mockedEventBridgeRuleService{
			ListEventBusesOutput: eventbridge.ListEventBusesOutput{
				EventBuses: []types.EventBus{
					{
						Arn:          aws.String(fmt.Sprintf("arn::%s", busName)),
						CreationTime: &now,
						Name:         aws.String(busName),
					},
				},
			},
			ListRulesOutput: eventbridge.ListRulesOutput{
				Rules: []types.Rule{
					{
						EventBusName: aws.String(busName),
						Name:         aws.String(rule1),
					},
					{
						EventBusName: aws.String(busName),
						Name:         aws.String(rule2),
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
			expected:  []string{bRule1, bRule2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(rule1),
					}},
				}},
			expected: []string{bRule2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			buses, err := service.getAll(
				context.Background(),
				config.Config{EventBridgeRule: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(buses))
		})
	}
}

func Test_EventBridgeRule_NukeAll(t *testing.T) {
	t.Parallel()

	ruleName := "bus-name|test-rule-1"
	service := EventBridgeRule{
		Client: mockedEventBridgeRuleService{
			DeleteRuleOutput: eventbridge.DeleteRuleOutput{},
		},
	}

	err := service.nukeAll([]*string{&ruleName})
	assert.NoError(t, err)
}
