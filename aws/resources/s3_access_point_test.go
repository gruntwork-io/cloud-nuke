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
	"github.com/stretchr/testify/require"
)

// mockS3AccessPointClient implements S3ControlAccessPointAPI for testing.
type mockS3AccessPointClient struct {
	ListAccessPointsOutput  s3control.ListAccessPointsOutput
	DeleteAccessPointOutput s3control.DeleteAccessPointOutput
	DeletedAccessPoints     []string
	CreationTimes           map[string]time.Time
	GetCalls                int
}

func (m *mockS3AccessPointClient) GetAccessPoint(ctx context.Context, params *s3control.GetAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.GetAccessPointOutput, error) {
	m.GetCalls++
	createdAt, ok := m.CreationTimes[aws.ToString(params.Name)]
	if !ok {
		return nil, errors.New("access denied")
	}
	return &s3control.GetAccessPointOutput{CreationDate: aws.Time(createdAt)}, nil
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

	ctx := context.Background()

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
			names, err := listS3AccessPoints(ctx, mock, resource.Scope{Region: "us-east-1", AccountID: "123456789012"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestS3AccessPoint_ListTimeFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ctx := context.Background()
	newMock := func() *mockS3AccessPointClient {
		return &mockS3AccessPointClient{
			ListAccessPointsOutput: s3control.ListAccessPointsOutput{
				AccessPointList: []types.AccessPoint{
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
	names, err := listS3AccessPoints(ctx, mock, resource.Scope{Region: "us-east-1", AccountID: "123456789012"}, olderThanAnHour)
	require.NoError(t, err)
	require.Equal(t, []string{"old"}, aws.ToStringSlice(names), "an unreadable creation time leaves the access point out")

	mock = newMock()
	olderThanAnHourExceptNew := olderThanAnHour
	olderThanAnHourExceptNew.ExcludeRule.NamesRegExp = []config.Expression{{RE: *regexp.MustCompile("^new$")}}
	_, err = listS3AccessPoints(ctx, mock, resource.Scope{Region: "us-east-1", AccountID: "123456789012"}, olderThanAnHourExceptNew)
	require.NoError(t, err)
	require.Equal(t, 2, mock.GetCalls, "an access point excluded by name gets no GetAccessPoint call")

	mock = newMock()
	names, err = listS3AccessPoints(ctx, mock, resource.Scope{Region: "us-east-1", AccountID: "123456789012"}, config.ResourceType{})
	require.NoError(t, err)
	require.Len(t, names, 3)
	require.Zero(t, mock.GetCalls, "no time filter, no GetAccessPoint calls")
}

func TestS3AccessPoint_Nuke(t *testing.T) {
	t.Parallel()

	mock := &mockS3AccessPointClient{}
	ctx := context.Background()

	results := nukeS3AccessPoints(
		ctx,
		mock,
		resource.Scope{Region: "us-east-1", AccountID: "123456789012"},
		"s3-access-point",
		[]*string{aws.String("test-access-point")},
	)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Equal(t, "test-access-point", results[0].Identifier)
	require.Contains(t, mock.DeletedAccessPoints, "test-access-point")
}

// TestS3AccessPoint_NukeAfterScan runs the resource the way the engine does:
// the provider hands the account id to Init, and neither the scan context nor
// the nuke context carries it.
func TestS3AccessPoint_NukeAfterScan(t *testing.T) {
	t.Parallel()

	mock := &mockS3AccessPointClient{
		ListAccessPointsOutput: s3control.ListAccessPointsOutput{
			AccessPointList: []types.AccessPoint{{Name: aws.String("test-access-point")}},
		},
	}
	r := NewS3AccessPoints().(*AwsResourceAdapter[S3ControlAccessPointAPI])
	r.Init(aws.Config{Region: "us-east-1"}, "123456789012")
	r.Client = mock

	ids, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)

	_, err = r.Nuke(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, []string{"test-access-point"}, mock.DeletedAccessPoints)
}

func TestS3AccessPoint_ListRequiresAccountID(t *testing.T) {
	t.Parallel()

	mock := &mockS3AccessPointClient{}
	// A scope without an account id is an error.
	_, err := listS3AccessPoints(context.Background(), mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to lookup the account id")
}
