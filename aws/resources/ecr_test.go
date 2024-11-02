package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedECR struct {
	ECRAPI
	DescribeRepositoriesOutput ecr.DescribeRepositoriesOutput
	DeleteRepositoryOutput     ecr.DeleteRepositoryOutput
}

func (m mockedECR) DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	return &m.DescribeRepositoriesOutput, nil
}

func (m mockedECR) DeleteRepository(ctx context.Context, params *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.DeleteRepositoryOutput, error) {
	return &m.DeleteRepositoryOutput, nil
}

func TestECR_GetAll(t *testing.T) {
	t.Parallel()
	testName1 := "test-repo1"
	testName2 := "test-repo2"
	now := time.Now()
	er := ECR{
		Client: &mockedECR{
			DescribeRepositoriesOutput: ecr.DescribeRepositoriesOutput{
				Repositories: []types.Repository{
					{
						RepositoryName: aws.String(testName1),
						CreatedAt:      aws.Time(now),
					},
					{
						RepositoryName: aws.String(testName2),
						CreatedAt:      aws.Time(now.Add(1)),
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
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := er.getAll(context.Background(), config.Config{
				ECRRepository: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}
func TestECR_NukeAll(t *testing.T) {
	t.Parallel()
	er := ECR{
		Client: &mockedECR{
			DeleteRepositoryOutput: ecr.DeleteRepositoryOutput{},
		},
	}

	err := er.nukeAll([]string{"test-repo1", "test-repo2"})
	require.NoError(t, err)
}
