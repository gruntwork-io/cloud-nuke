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

type mocks3ObjectLambdaAccessPoint struct {
	s3controliface.S3ControlAPI
	ListAccessPointsForObjectLambdaOutput  s3control.ListAccessPointsForObjectLambdaOutput
	DeleteAccessPointForObjectLambdaOutput s3control.DeleteAccessPointForObjectLambdaOutput
}

func (m mocks3ObjectLambdaAccessPoint) ListAccessPointsForObjectLambdaPages(_ *s3control.ListAccessPointsForObjectLambdaInput, fn func(*s3control.ListAccessPointsForObjectLambdaOutput, bool) bool) error {
	fn(&m.ListAccessPointsForObjectLambdaOutput, true)
	return nil
}
func (m mocks3ObjectLambdaAccessPoint) DeleteAccessPointForObjectLambda(_ *s3control.DeleteAccessPointForObjectLambdaInput) (*s3control.DeleteAccessPointForObjectLambdaOutput, error) {
	return &m.DeleteAccessPointForObjectLambdaOutput, nil
}

func TestS3ObjectLambdaAccessPoint_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName01 := "test-access-point-01"
	testName02 := "test-access-point-02"

	ctx := context.Background()
	ctx = context.WithValue(ctx, util.AccountIdKey, "test-account-id")

	ap := S3ObjectLambdaAccessPoint{
		Client: mocks3ObjectLambdaAccessPoint{
			ListAccessPointsForObjectLambdaOutput: s3control.ListAccessPointsForObjectLambdaOutput{
				ObjectLambdaAccessPointList: []*s3control.ObjectLambdaAccessPoint{
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
				S3ObjectLambdaAccessPoint: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestS3ObjectLambdaAccessPoint_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	rc := S3ObjectLambdaAccessPoint{
		Client: mocks3ObjectLambdaAccessPoint{
			DeleteAccessPointForObjectLambdaOutput: s3control.DeleteAccessPointForObjectLambdaOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
