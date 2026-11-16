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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockS3MultiRegionAccessPointClient struct {
	ListOutput   s3control.ListMultiRegionAccessPointsOutput
	DeleteOutput s3control.DeleteMultiRegionAccessPointOutput
	DeleteErr    error
	Deleted      []string
}

func (m *mockS3MultiRegionAccessPointClient) ListMultiRegionAccessPoints(_ context.Context, _ *s3control.ListMultiRegionAccessPointsInput, _ ...func(*s3control.Options)) (*s3control.ListMultiRegionAccessPointsOutput, error) {
	return &m.ListOutput, nil
}

func (m *mockS3MultiRegionAccessPointClient) DeleteMultiRegionAccessPoint(_ context.Context, params *s3control.DeleteMultiRegionAccessPointInput, _ ...func(*s3control.Options)) (*s3control.DeleteMultiRegionAccessPointOutput, error) {
	if params.Details != nil {
		m.Deleted = append(m.Deleted, aws.ToString(params.Details.Name))
	}
	return &m.DeleteOutput, m.DeleteErr
}

func TestS3MultiRegionAccessPoint_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName01 := "test-access-point-01"
	testName02 := "test-access-point-02"

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
					}},
				},
			},
			expected: []string{testName02},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{testName01},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")
			client := &mockS3MultiRegionAccessPointClient{
				ListOutput: s3control.ListMultiRegionAccessPointsOutput{
					AccessPoints: []types.MultiRegionAccessPointReport{
						{Name: aws.String(testName01), CreatedAt: aws.Time(now)},
						{Name: aws.String(testName02), CreatedAt: aws.Time(now.Add(1 * time.Second))},
					},
				},
			}

			names, err := listS3MultiRegionAccessPoints(ctx, client, resource.Scope{Region: "global"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestS3MultiRegionAccessPoint_Nuke(t *testing.T) {
	t.Parallel()

	client := &mockS3MultiRegionAccessPointClient{
		DeleteOutput: s3control.DeleteMultiRegionAccessPointOutput{},
	}

	ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")
	results := nukeS3MultiRegionAccessPoints(
		ctx,
		client,
		resource.Scope{Region: "global"},
		"s3-multi-region-access-point",
		[]*string{aws.String("test-access-point")},
	)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Equal(t, "test-access-point", results[0].Identifier)
}

// TestS3MultiRegionAccessPoint_NukeAfterScan runs the resource the way the
// engine does: the scan context carries the account id, and the nuke context does not.
func TestS3MultiRegionAccessPoint_NukeAfterScan(t *testing.T) {
	t.Parallel()

	client := &mockS3MultiRegionAccessPointClient{
		ListOutput: s3control.ListMultiRegionAccessPointsOutput{
			AccessPoints: []types.MultiRegionAccessPointReport{{Name: aws.String("test-access-point")}},
		},
	}
	r := NewS3MultiRegionAccessPoints().(*AwsResourceAdapter[S3ControlMultiRegionAPI]).Resource
	r.Client = client

	scanCtx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")
	ids, err := r.GetAndSetIdentifiers(scanCtx, config.Config{})
	require.NoError(t, err)

	_, err = r.Nuke(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, []string{"test-access-point"}, client.Deleted)
}
