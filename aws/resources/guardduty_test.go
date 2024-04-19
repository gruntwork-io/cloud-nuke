package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/aws/aws-sdk-go/service/guardduty/guarddutyiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type mockedGuardDuty struct {
	guarddutyiface.GuardDutyAPI
	ListDetectorsPagesOutput guardduty.ListDetectorsOutput
	GetDetectorOutput        guardduty.GetDetectorOutput
	DeleteDetectorOutput     guardduty.DeleteDetectorOutput
}

func (m mockedGuardDuty) ListDetectorsPages(input *guardduty.ListDetectorsInput, callback func(*guardduty.ListDetectorsOutput, bool) bool) error {
	callback(&m.ListDetectorsPagesOutput, true)
	return nil
}

func (m mockedGuardDuty) GetDetector(input *guardduty.GetDetectorInput) (*guardduty.GetDetectorOutput, error) {
	return &m.GetDetectorOutput, nil
}

func (m mockedGuardDuty) DeleteDetector(input *guardduty.DeleteDetectorInput) (*guardduty.DeleteDetectorOutput, error) {
	return &m.DeleteDetectorOutput, nil
}

func TestGuardDuty_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "test-detector-id-1"
	testId2 := "test-detector-id-2"
	now := time.Now()
	gd := GuardDuty{
		Client: mockedGuardDuty{
			ListDetectorsPagesOutput: guardduty.ListDetectorsOutput{
				DetectorIds: []*string{
					aws.String(testId1),
					aws.String(testId2),
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
