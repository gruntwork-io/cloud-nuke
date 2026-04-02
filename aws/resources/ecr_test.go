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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockECRClient struct {
	DescribeRepositoriesOutput ecr.DescribeRepositoriesOutput
	DeleteRepositoryOutput     ecr.DeleteRepositoryOutput
}

func (m *mockECRClient) DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	return &m.DescribeRepositoriesOutput, nil
}

func (m *mockECRClient) DeleteRepository(ctx context.Context, params *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.DeleteRepositoryOutput, error) {
	return &m.DeleteRepositoryOutput, nil
}

func (m *mockECRClient) ListTagsForResource(ctx context.Context, params *ecr.ListTagsForResourceInput, optFns ...func(*ecr.Options)) (*ecr.ListTagsForResourceOutput, error) {
	if aws.ToString(params.ResourceArn) == "arn:aws:ecr:us-east-1:123456789:repository/test-repo2" {
		return &ecr.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}}, nil
	}
	return &ecr.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}}}, nil
}

func TestListECRRepositories(t *testing.T) {
	t.Parallel()

	testName1 := "test-repo1"
	testName2 := "test-repo2"
	now := time.Now()

	mock := &mockECRClient{
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
	}

	names, err := listECRRepositories(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testName1, testName2}, aws.ToStringSlice(names))
}

func TestListECRRepositories_WithFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-repo1"
	testName2 := "skip-this"
	now := time.Now()

	mock := &mockECRClient{
		DescribeRepositoriesOutput: ecr.DescribeRepositoriesOutput{
			Repositories: []types.Repository{
				{
					RepositoryName: aws.String(testName1),
					CreatedAt:      aws.Time(now),
				},
				{
					RepositoryName: aws.String(testName2),
					CreatedAt:      aws.Time(now),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listECRRepositories(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testName1}, aws.ToStringSlice(names))
}

func TestListECRRepositories_TimeFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-repo1"
	testName2 := "test-repo2"
	now := time.Now()

	mock := &mockECRClient{
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
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now),
		},
	}

	names, err := listECRRepositories(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testName1}, aws.ToStringSlice(names))
}

func TestListECRRepositories_TagInclusionFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-repo1"
	testName2 := "test-repo2"
	now := time.Now()

	mock := &mockECRClient{
		DescribeRepositoriesOutput: ecr.DescribeRepositoriesOutput{
			Repositories: []types.Repository{
				{
					RepositoryName: aws.String(testName1),
					RepositoryArn:  aws.String("arn:aws:ecr:us-east-1:123456789:repository/test-repo1"),
					CreatedAt:      aws.Time(now),
				},
				{
					RepositoryName: aws.String(testName2),
					RepositoryArn:  aws.String("arn:aws:ecr:us-east-1:123456789:repository/test-repo2"),
					CreatedAt:      aws.Time(now),
				},
			},
		},
	}

	cfg := config.ResourceType{
		IncludeRule: config.FilterRule{
			Tags: map[string]config.Expression{
				"env": {RE: *regexp.MustCompile("^prod$")},
			},
		},
	}

	names, err := listECRRepositories(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testName2}, aws.ToStringSlice(names))
}

func TestDeleteECRRepository(t *testing.T) {
	t.Parallel()

	mock := &mockECRClient{}
	err := deleteECRRepository(context.Background(), mock, aws.String("test-repo"))
	require.NoError(t, err)
}
