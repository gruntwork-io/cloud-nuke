package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedGuardDuty struct {
	GuardDutyAPI
	ListDetectorsPagesOutput guardduty.ListDetectorsOutput
	GetDetectorOutput        guardduty.GetDetectorOutput
	DeleteDetectorOutput     guardduty.DeleteDetectorOutput
}

func (m mockedGuardDuty) GetDetector(ctx context.Context, params *guardduty.GetDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.GetDetectorOutput, error) {
	return &m.GetDetectorOutput, nil
}

func (m mockedGuardDuty) DeleteDetector(ctx context.Context, params *guardduty.DeleteDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.DeleteDetectorOutput, error) {
	return &m.DeleteDetectorOutput, nil
}

func (m mockedGuardDuty) ListDetectors(ctx context.Context, params *guardduty.ListDetectorsInput, optFns ...func(*guardduty.Options)) (*guardduty.ListDetectorsOutput, error) {
	return &m.ListDetectorsPagesOutput, nil
}

func TestGuardDuty_GetAll(t *testing.T) {
	t.Parallel()
	testId1 := "test-detector-id-1"
	testId2 := "test-detector-id-2"
	now := time.Now()
	gd := GuardDuty{
		Client: mockedGuardDuty{
			ListDetectorsPagesOutput: guardduty.ListDetectorsOutput{
				DetectorIds: []string{
					testId1,
					testId2,
				},
			},
			GetDetectorOutput: guardduty.GetDetectorOutput{
				CreatedAt: aws.String(now.Format(time.RFC3339)),
			},
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
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := gd.getAll(context.Background(), config.Config{
				GuardDuty: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestGuardDuty_NukeAll(t *testing.T) {
	t.Parallel()
	gd := GuardDuty{
		Client: mockedGuardDuty{
			DeleteDetectorOutput: guardduty.DeleteDetectorOutput{},
		},
	}

	err := gd.nukeAll([]string{"test"})
	require.NoError(t, err)
}
