package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

// mockS3AccessPointClient implements S3ControlAccessPointAPI for testing.
type mockS3AccessPointClient struct {
	ListAccessPointsOutput  s3control.ListAccessPointsOutput
	DeleteAccessPointOutput s3control.DeleteAccessPointOutput
	DeletedAccessPoints     []string
}

func (m *mockS3AccessPointClient) ListAccessPoints(ctx context.Context, params *s3control.ListAccessPointsInput, optFns ...func(*s3control.Options)) (*s3control.ListAccessPointsOutput, error) {
	return &m.ListAccessPointsOutput, nil
}

func (m *mockS3AccessPointClient) DeleteAccessPoint(ctx context.Context, params *s3control.DeleteAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointOutput, error) {
	m.DeletedAccessPoints = append(m.DeletedAccessPoints, aws.ToString(params.Name))
	return &m.DeleteAccessPointOutput, nil
}

func TestS3AccessPoint_List(t *testing.T) {
	t.Parallel()

	testName1 := "test-access-point-01"
	testName2 := "test-access-point-02"

	mock := &mockS3AccessPointClient{
		ListAccessPointsOutput: s3control.ListAccessPointsOutput{
			AccessPointList: []types.AccessPoint{
				{Name: aws.String(testName1)},
				{Name: aws.String(testName2)},
			},
		},
	}

	ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listS3AccessPoints(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestS3AccessPoint_Nuke(t *testing.T) {
	t.Parallel()

	mock := &mockS3AccessPointClient{}
	ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")

	results := nukeS3AccessPoints(
		ctx,
		mock,
		resource.Scope{Region: "us-east-1"},
		"s3-ap",
		[]*string{aws.String("test-access-point")},
	)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Equal(t, "test-access-point", results[0].Identifier)
	require.Contains(t, mock.DeletedAccessPoints, "test-access-point")
}

func TestS3AccessPoint_ListRequiresAccountID(t *testing.T) {
	t.Parallel()

	mock := &mockS3AccessPointClient{}
	// Context without account ID should return error
	_, err := listS3AccessPoints(context.Background(), mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to lookup the account id")
}
