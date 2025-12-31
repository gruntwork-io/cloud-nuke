package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockGuardDutyClient struct {
	ListDetectorsOutput  guardduty.ListDetectorsOutput
	GetDetectorOutput    guardduty.GetDetectorOutput
	DeleteDetectorOutput guardduty.DeleteDetectorOutput
}

func (m *mockGuardDutyClient) GetDetector(ctx context.Context, params *guardduty.GetDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.GetDetectorOutput, error) {
	return &m.GetDetectorOutput, nil
}

func (m *mockGuardDutyClient) DeleteDetector(ctx context.Context, params *guardduty.DeleteDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.DeleteDetectorOutput, error) {
	return &m.DeleteDetectorOutput, nil
}

func (m *mockGuardDutyClient) ListDetectors(ctx context.Context, params *guardduty.ListDetectorsInput, optFns ...func(*guardduty.Options)) (*guardduty.ListDetectorsOutput, error) {
	return &m.ListDetectorsOutput, nil
}

func TestListGuardDutyDetectors(t *testing.T) {
	t.Parallel()

	testId1 := "test-detector-id-1"
	testId2 := "test-detector-id-2"
	now := time.Now()

	mock := &mockGuardDutyClient{
		ListDetectorsOutput: guardduty.ListDetectorsOutput{
			DetectorIds: []string{testId1, testId2},
		},
		GetDetectorOutput: guardduty.GetDetectorOutput{
			CreatedAt: aws.String(now.Format(time.RFC3339)),
		},
	}

	ids, err := listGuardDutyDetectors(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testId1, testId2}, aws.ToStringSlice(ids))
}

func TestListGuardDutyDetectors_TimeFilter(t *testing.T) {
	t.Parallel()

	testId1 := "test-detector-id-1"
	now := time.Now()

	mock := &mockGuardDutyClient{
		ListDetectorsOutput: guardduty.ListDetectorsOutput{
			DetectorIds: []string{testId1},
		},
		GetDetectorOutput: guardduty.GetDetectorOutput{
			CreatedAt: aws.String(now.Format(time.RFC3339)),
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-2 * time.Hour)),
		},
	}

	ids, err := listGuardDutyDetectors(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestDeleteGuardDutyDetector(t *testing.T) {
	t.Parallel()

	mock := &mockGuardDutyClient{}
	err := deleteGuardDutyDetector(context.Background(), mock, aws.String("test-detector-id"))
	require.NoError(t, err)
}
