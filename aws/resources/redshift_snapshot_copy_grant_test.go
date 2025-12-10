package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedRedshiftSnapshotCopyGrants struct {
	RedshiftSnapshotCopyGrantsAPI

	DescribeSnapshotCopyGrantsOutput redshift.DescribeSnapshotCopyGrantsOutput
	DeleteSnapshotCopyGrantOutput    redshift.DeleteSnapshotCopyGrantOutput
}

func (m mockedRedshiftSnapshotCopyGrants) DescribeSnapshotCopyGrants(ctx context.Context, input *redshift.DescribeSnapshotCopyGrantsInput, opts ...func(*redshift.Options)) (*redshift.DescribeSnapshotCopyGrantsOutput, error) {
	return &m.DescribeSnapshotCopyGrantsOutput, nil
}

func (m mockedRedshiftSnapshotCopyGrants) DeleteSnapshotCopyGrant(ctx context.Context, input *redshift.DeleteSnapshotCopyGrantInput, opts ...func(*redshift.Options)) (*redshift.DeleteSnapshotCopyGrantOutput, error) {
	return &m.DeleteSnapshotCopyGrantOutput, nil
}

func TestRedshiftSnapshotCopyGrant_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-grant1"
	testName2 := "test-grant2"
	g := RedshiftSnapshotCopyGrants{
		Client: mockedRedshiftSnapshotCopyGrants{
			DescribeSnapshotCopyGrantsOutput: redshift.DescribeSnapshotCopyGrantsOutput{
				SnapshotCopyGrants: []types.SnapshotCopyGrant{
					{
						SnapshotCopyGrantName: aws.String(testName1),
					},
					{
						SnapshotCopyGrantName: aws.String(testName2),
					},
				},
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
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := g.getAll(context.Background(), config.Config{
				RedshiftSnapshotCopyGrant: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestRedshiftSnapshotCopyGrant_NukeAll(t *testing.T) {
	t.Parallel()

	g := RedshiftSnapshotCopyGrants{
		Client: mockedRedshiftSnapshotCopyGrants{
			DeleteSnapshotCopyGrantOutput: redshift.DeleteSnapshotCopyGrantOutput{},
		},
	}
	g.Context = context.Background()

	err := g.nukeAll([]*string{aws.String("test-grant")})
	require.NoError(t, err)
}
