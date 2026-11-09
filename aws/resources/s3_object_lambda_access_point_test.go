package resources

import (
	"context"
	"errors"
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

type mockS3ObjectLambdaAccessPointClient struct {
	ListAccessPointsForObjectLambdaOutput  s3control.ListAccessPointsForObjectLambdaOutput
	DeleteAccessPointForObjectLambdaOutput s3control.DeleteAccessPointForObjectLambdaOutput
	DeletedAccessPoints                    []string
	CreationTimes                          map[string]time.Time
	GetCalls                               int
}

func (m *mockS3ObjectLambdaAccessPointClient) GetAccessPointForObjectLambda(ctx context.Context, params *s3control.GetAccessPointForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.GetAccessPointForObjectLambdaOutput, error) {
	m.GetCalls++
	createdAt, ok := m.CreationTimes[aws.ToString(params.Name)]
	if !ok {
		return nil, errors.New("access denied")
	}
	return &s3control.GetAccessPointForObjectLambdaOutput{CreationDate: aws.Time(createdAt)}, nil
}

func (m *mockS3ObjectLambdaAccessPointClient) ListAccessPointsForObjectLambda(ctx context.Context, params *s3control.ListAccessPointsForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.ListAccessPointsForObjectLambdaOutput, error) {
	return &m.ListAccessPointsForObjectLambdaOutput, nil
}

func (m *mockS3ObjectLambdaAccessPointClient) DeleteAccessPointForObjectLambda(ctx context.Context, params *s3control.DeleteAccessPointForObjectLambdaInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointForObjectLambdaOutput, error) {
	m.DeletedAccessPoints = append(m.DeletedAccessPoints, aws.ToString(params.Name))
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

func TestS3ObjectLambdaAccessPoint_ListTimeFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")
	newMock := func() *mockS3ObjectLambdaAccessPointClient {
		return &mockS3ObjectLambdaAccessPointClient{
			ListAccessPointsForObjectLambdaOutput: s3control.ListAccessPointsForObjectLambdaOutput{
				ObjectLambdaAccessPointList: []types.ObjectLambdaAccessPoint{
					{Name: aws.String("old")},
					{Name: aws.String("new")},
					{Name: aws.String("unreadable")},
				},
			},
			CreationTimes: map[string]time.Time{
				"old": now.Add(-2 * time.Hour),
				"new": now,
			},
		}
	}

	mock := newMock()
	olderThanAnHour := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-time.Hour))}}
	names, err := listS3ObjectLambdaAccessPoints(ctx, mock, resource.Scope{Region: "us-east-1"}, olderThanAnHour)
	require.NoError(t, err)
	require.Equal(t, []string{"old"}, aws.ToStringSlice(names), "an unreadable creation time leaves the access point out")

	mock = newMock()
	olderThanAnHourExceptNew := olderThanAnHour
	olderThanAnHourExceptNew.ExcludeRule.NamesRegExp = []config.Expression{{RE: *regexp.MustCompile("^new$")}}
	_, err = listS3ObjectLambdaAccessPoints(ctx, mock, resource.Scope{Region: "us-east-1"}, olderThanAnHourExceptNew)
	require.NoError(t, err)
	require.Equal(t, 2, mock.GetCalls, "an access point excluded by name gets no GetAccessPointForObjectLambda call")

	mock = newMock()
	names, err = listS3ObjectLambdaAccessPoints(ctx, mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Len(t, names, 3)
	require.Zero(t, mock.GetCalls, "no time filter, no GetAccessPointForObjectLambda calls")
}

func TestS3ObjectLambdaAccessPoint_Nuke(t *testing.T) {
	t.Parallel()

	mock := &mockS3ObjectLambdaAccessPointClient{}
	ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")

	results := nukeS3ObjectLambdaAccessPoints(
		ctx,
		mock,
		resource.Scope{Region: "us-east-1"},
		"s3-object-lambda-access-point",
		[]*string{aws.String("test-access-point")},
	)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Equal(t, "test-access-point", results[0].Identifier)
}

// TestS3ObjectLambdaAccessPoint_NukeAfterScan runs the resource the way the
// engine does: the scan context carries the account id, and the nuke context does not.
func TestS3ObjectLambdaAccessPoint_NukeAfterScan(t *testing.T) {
	t.Parallel()

	mock := &mockS3ObjectLambdaAccessPointClient{
		ListAccessPointsForObjectLambdaOutput: s3control.ListAccessPointsForObjectLambdaOutput{
			ObjectLambdaAccessPointList: []types.ObjectLambdaAccessPoint{{Name: aws.String("test-access-point")}},
		},
	}
	r := NewS3ObjectLambdaAccessPoints().(*AwsResourceAdapter[S3ObjectLambdaAccessPointAPI]).Resource
	r.Client = mock

	scanCtx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")
	ids, err := r.GetAndSetIdentifiers(scanCtx, config.Config{})
	require.NoError(t, err)

	_, err = r.Nuke(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, []string{"test-access-point"}, mock.DeletedAccessPoints)
}

func TestS3ObjectLambdaAccessPoint_ListRequiresAccountID(t *testing.T) {
	t.Parallel()

	mock := &mockS3ObjectLambdaAccessPointClient{}
	// Context without account ID should return error
	_, err := listS3ObjectLambdaAccessPoints(context.Background(), mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to lookup the account id")
}
