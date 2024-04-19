package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/aws/aws-sdk-go/service/s3control/s3controliface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mocks3AccessPoint struct {
	s3controliface.S3ControlAPI
	ListAccessPointsOutput  s3control.ListAccessPointsOutput
	DeleteAccessPointOutput s3control.DeleteAccessPointOutput
}

func (m mocks3AccessPoint) ListAccessPointsPages(_ *s3control.ListAccessPointsInput, fn func(*s3control.ListAccessPointsOutput, bool) bool) error {
	fn(&m.ListAccessPointsOutput, true)
	return nil
}
func (m mocks3AccessPoint) DeleteAccessPoint(_ *s3control.DeleteAccessPointInput) (*s3control.DeleteAccessPointOutput, error) {
	return &m.DeleteAccessPointOutput, nil
}

func TestS3AccessPoint_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName01 := "test-access-point-01"
	testName02 := "test-access-point-02"

	ctx := context.Background()
	ctx = context.WithValue(ctx, util.AccountIdKey, "test-account-id")

	ap := S3AccessPoint{
		Client: mocks3AccessPoint{
			ListAccessPointsOutput: s3control.ListAccessPointsOutput{
				AccessPointList: []*s3control.AccessPoint{
					{
						Name: aws.String(testName01),
					},
					{
						Name: aws.String(testName02),
					},
				},
			},
		},
		AccountID: aws.String("test-account-id"),
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName01, testName02},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName01),
					}}},
			},
			expected: []string{testName02},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			names, err := ap.getAll(ctx, config.Config{
				S3AccessPoint: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestS3AccessPoint_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	rc := S3AccessPoint{
		Client: mocks3AccessPoint{
			DeleteAccessPointOutput: s3control.DeleteAccessPointOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
