package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockS3MultiRegionAccessPoint struct {
	S3ControlMultiRegionAPI
	ListMultiRegionAccessPointsOutput  s3control.ListMultiRegionAccessPointsOutput
	DeleteMultiRegionAccessPointOutput s3control.DeleteMultiRegionAccessPointOutput
}

func (m mockS3MultiRegionAccessPoint) ListMultiRegionAccessPoints(ctx context.Context, params *s3control.ListMultiRegionAccessPointsInput, optFns ...func(*s3control.Options)) (*s3control.ListMultiRegionAccessPointsOutput, error) {
	return &m.ListMultiRegionAccessPointsOutput, nil
}
func (m mockS3MultiRegionAccessPoint) DeleteMultiRegionAccessPoint(ctx context.Context, params *s3control.DeleteMultiRegionAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteMultiRegionAccessPointOutput, error) {
	return &m.DeleteMultiRegionAccessPointOutput, nil
}

func TestS3MultiRegionAccessPoint_GetAll(t *testing.T) {

	t.Parallel()

	testName01 := "test-access-point-01"
	testName02 := "test-access-point-02"

	ctx := context.Background()
	ctx = context.WithValue(ctx, util.AccountIdKey, "test-account-id")

	now := time.Now()

	ap := S3MultiRegionAccessPoint{
		Region: "us-west-2",
		Client: mockS3MultiRegionAccessPoint{
			ListMultiRegionAccessPointsOutput: s3control.ListMultiRegionAccessPointsOutput{
				AccessPoints: []types.MultiRegionAccessPointReport{
					{
						Name:      aws.String(testName01),
						CreatedAt: aws.Time(now),
					},
					{
						Name:      aws.String(testName02),
						CreatedAt: aws.Time(now.Add(1)),
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
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName01},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			names, err := ap.getAll(ctx, config.Config{
				S3MultiRegionAccessPoint: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestS3MultiRegionAccessPoint_NukeAll(t *testing.T) {

	t.Parallel()

	rc := S3MultiRegionAccessPoint{
		Region: "us-west-2",
		Client: mockS3MultiRegionAccessPoint{
			DeleteMultiRegionAccessPointOutput: s3control.DeleteMultiRegionAccessPointOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
