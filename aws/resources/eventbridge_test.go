package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedEventBridgeService struct {
	EventBridgeAPI
	DeleteEventBusOutput eventbridge.DeleteEventBusOutput
	ListEventBusesOutput eventbridge.ListEventBusesOutput
}

func (m mockedEventBridgeService) DeleteEventBus(ctx context.Context, params *eventbridge.DeleteEventBusInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteEventBusOutput, error) {
	return &m.DeleteEventBusOutput, nil
}

func (m mockedEventBridgeService) ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error) {
	return &m.ListEventBusesOutput, nil
}

func Test_EventBridge_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()

	bus1 := "test-bus-1"
	bus2 := "test-bus-2"

	service := EventBridge{
		Client: mockedEventBridgeService{
			ListEventBusesOutput: eventbridge.ListEventBusesOutput{
				EventBuses: []types.EventBus{
					{
						Arn:          aws.String(fmt.Sprintf("arn::%s", bus1)),
						CreationTime: &now,
						Name:         aws.String(bus1),
					},
					{
						Arn:          aws.String(fmt.Sprintf("arn::%s", bus2)),
						CreationTime: aws.Time(now.Add(time.Hour)),
						Name:         aws.String(bus2),
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
			expected:  []string{bus1, bus2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(bus1),
					}},
				}},
			expected: []string{bus2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			buses, err := service.getAll(
				context.Background(),
				config.Config{EventBridge: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(buses))
		})
	}
}

func Test_EventBridge_NukeAll(t *testing.T) {
	t.Parallel()

	busName := "test-bus-1"
	service := EventBridge{
		Client: mockedEventBridgeService{
			DeleteEventBusOutput: eventbridge.DeleteEventBusOutput{},
		},
	}

	err := service.nukeAll([]*string{&busName})
	assert.NoError(t, err)
}
