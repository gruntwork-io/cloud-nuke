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

type mockS3ObjectLambdaAccessPointClient struct {
	ListAccessPointsForObjectLambdaOutput  s3control.ListAccessPointsForObjectLambdaOutput
	DeleteAccessPointForObjectLambdaOutput s3control.DeleteAccessPointForObjectLambdaOutput
}

func (m *mockS3ObjectLambdaAccessPointClient) ListAccessPointsForObjectLambda(ctx context.Context, params *s3control.ListAccessPointsForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.ListAccessPointsForObjectLambdaOutput, error) {
	return &m.ListAccessPointsForObjectLambdaOutput, nil
}

func (m *mockS3ObjectLambdaAccessPointClient) DeleteAccessPointForObjectLambda(ctx context.Context, params *s3control.DeleteAccessPointForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointForObjectLambdaOutput, error) {
	return &m.DeleteAccessPointForObjectLambdaOutput, nil
}

func TestS3ObjectLambdaAccessPoint_List(t *testing.T) {
	t.Parallel()

	testName1 := "test-access-point-01"
	testName2 := "test-access-point-02"

	mock := &mockS3ObjectLambdaAccessPointClient{
		ListAccessPointsForObjectLambdaOutput: s3control.ListAccessPointsForObjectLambdaOutput{
			ObjectLambdaAccessPointList: []types.ObjectLambdaAccessPoint{
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
			names, err := listS3ObjectLambdaAccessPoints(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestS3ObjectLambdaAccessPoint_Nuke(t *testing.T) {
	t.Parallel()

	mock := &mockS3ObjectLambdaAccessPointClient{}
	ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")

	results := nukeS3ObjectLambdaAccessPoints(
		ctx,
		mock,
		resource.Scope{Region: "us-east-1"},
		"s3-olap",
		[]*string{aws.String("test-access-point")},
	)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Equal(t, "test-access-point", results[0].Identifier)
}

func TestS3ObjectLambdaAccessPoint_ListRequiresAccountID(t *testing.T) {
	t.Parallel()

	mock := &mockS3ObjectLambdaAccessPointClient{}
	// Context without account ID should return error
	_, err := listS3ObjectLambdaAccessPoints(context.Background(), mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to lookup the account id")
}
