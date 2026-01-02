package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockRedshiftSnapshotCopyGrantsClient struct {
	DescribeSnapshotCopyGrantsOutput redshift.DescribeSnapshotCopyGrantsOutput
	DeleteSnapshotCopyGrantOutput    redshift.DeleteSnapshotCopyGrantOutput
}

func (m *mockRedshiftSnapshotCopyGrantsClient) DescribeSnapshotCopyGrants(ctx context.Context, input *redshift.DescribeSnapshotCopyGrantsInput, opts ...func(*redshift.Options)) (*redshift.DescribeSnapshotCopyGrantsOutput, error) {
	return &m.DescribeSnapshotCopyGrantsOutput, nil
}

func (m *mockRedshiftSnapshotCopyGrantsClient) DeleteSnapshotCopyGrant(ctx context.Context, input *redshift.DeleteSnapshotCopyGrantInput, opts ...func(*redshift.Options)) (*redshift.DeleteSnapshotCopyGrantOutput, error) {
	return &m.DeleteSnapshotCopyGrantOutput, nil
}

func TestListRedshiftSnapshotCopyGrants(t *testing.T) {
	t.Parallel()

	testName1 := "test-grant1"
	testName2 := "test-grant2"

	mock := &mockRedshiftSnapshotCopyGrantsClient{
		DescribeSnapshotCopyGrantsOutput: redshift.DescribeSnapshotCopyGrantsOutput{
			SnapshotCopyGrants: []types.SnapshotCopyGrant{
				{SnapshotCopyGrantName: aws.String(testName1)},
				{SnapshotCopyGrantName: aws.String(testName2)},
			},
		},
	}

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
			names, err := listRedshiftSnapshotCopyGrants(context.Background(), mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteRedshiftSnapshotCopyGrant(t *testing.T) {
	t.Parallel()

	mock := &mockRedshiftSnapshotCopyGrantsClient{}
	err := deleteRedshiftSnapshotCopyGrant(context.Background(), mock, aws.String("test-grant"))
	require.NoError(t, err)
}
