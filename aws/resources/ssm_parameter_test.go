package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSSMParameterClient implements SSMParameterAPI for testing.
type mockSSMParameterClient struct {
	DescribeParametersOutput ssm.DescribeParametersOutput
	DescribeParametersError  error

	// TagsByName maps parameter name to its tags. If a name is absent, an empty tag list is returned.
	TagsByName     map[string][]types.Tag
	TagsError      error
	TagsErrorNames map[string]bool // names that trigger TagsError

	DeleteParameterError error
}

func (m *mockSSMParameterClient) DescribeParameters(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	if m.DescribeParametersError != nil {
		return nil, m.DescribeParametersError
	}
	return &m.DescribeParametersOutput, nil
}

func (m *mockSSMParameterClient) ListTagsForResource(ctx context.Context, params *ssm.ListTagsForResourceInput, optFns ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error) {
	name := aws.ToString(params.ResourceId)
	if m.TagsErrorNames != nil && m.TagsErrorNames[name] {
		return nil, m.TagsError
	}
	return &ssm.ListTagsForResourceOutput{TagList: m.TagsByName[name]}, nil
}

func (m *mockSSMParameterClient) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	if m.DeleteParameterError != nil {
		return nil, m.DeleteParameterError
	}
	return &ssm.DeleteParameterOutput{}, nil
}

func TestSSMParameter_ResourceName(t *testing.T) {
	t.Parallel()
	r := NewSSMParameter()
	assert.Equal(t, "ssm-parameter", r.ResourceName())
}

func TestSSMParameter_MaxBatchSize(t *testing.T) {
	t.Parallel()
	r := NewSSMParameter()
	assert.Equal(t, 10, r.MaxBatchSize())
}

func TestSSMParameter_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()

	mock := &mockSSMParameterClient{
		DescribeParametersOutput: ssm.DescribeParametersOutput{
			Parameters: []types.ParameterMetadata{
				{Name: aws.String("/app/db-password"), LastModifiedDate: aws.Time(now)},
				{Name: aws.String("/app/api-key"), LastModifiedDate: aws.Time(now.Add(1 * time.Hour))},
				{Name: nil, LastModifiedDate: aws.Time(now)}, // nil names must be skipped
			},
		},
		TagsByName: map[string][]types.Tag{
			"/app/db-password": {{Key: aws.String("Env"), Value: aws.String("prod")}},
			"/app/api-key":     {{Key: aws.String("Env"), Value: aws.String("test")}},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"/app/db-password", "/app/api-key"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("db-password")}},
				},
			},
			expected: []string{"/app/api-key"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{"/app/db-password"},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{"/app/api-key"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			names, err := listSSMParameters(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

// TestSSMParameter_GetAll_TagFetchFailure verifies that a parameter whose tags cannot be
// fetched is skipped. Proceeding with nil tags would bypass the cloud-nuke-excluded and
// cloud-nuke-after protection checks, risking accidental deletion of protected resources.
func TestSSMParameter_GetAll_TagFetchFailure(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockSSMParameterClient{
		DescribeParametersOutput: ssm.DescribeParametersOutput{
			Parameters: []types.ParameterMetadata{
				{Name: aws.String("/param/good"), LastModifiedDate: aws.Time(now)},
				{Name: aws.String("/param/no-tag-access"), LastModifiedDate: aws.Time(now)},
			},
		},
		TagsError:      fmt.Errorf("AccessDenied: not authorized to list tags"),
		TagsErrorNames: map[string]bool{"/param/no-tag-access": true},
	}

	names, err := listSSMParameters(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"/param/good"}, aws.ToStringSlice(names))
}

// TestSSMParameter_GetAll_SkipsAwsManagedParameters verifies that AWS-managed public
// parameters (those under /aws/) are never included for deletion.
func TestSSMParameter_GetAll_SkipsAwsManagedParameters(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockSSMParameterClient{
		DescribeParametersOutput: ssm.DescribeParametersOutput{
			Parameters: []types.ParameterMetadata{
				{Name: aws.String("/app/my-param"), LastModifiedDate: aws.Time(now)},
				{Name: aws.String("/aws/service/ami-amazon-linux-latest/hvm/ebs/x86_64/al2/recommended/ami-id"), LastModifiedDate: aws.Time(now)},
				{Name: aws.String("/aws/reference/secretsmanager/my-secret"), LastModifiedDate: aws.Time(now)},
			},
		},
		TagsByName: map[string][]types.Tag{
			"/app/my-param": {},
		},
	}

	names, err := listSSMParameters(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"/app/my-param"}, aws.ToStringSlice(names))
}

func TestSSMParameter_GetAll_DescribeError(t *testing.T) {
	t.Parallel()

	mock := &mockSSMParameterClient{
		DescribeParametersError: fmt.Errorf("AccessDenied: not authorized"),
	}

	_, err := listSSMParameters(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "AccessDenied")
}

func TestSSMParameter_NukeAll(t *testing.T) {
	t.Parallel()

	mock := &mockSSMParameterClient{}
	err := deleteSSMParameter(context.Background(), mock, aws.String("/app/db-password"))
	require.NoError(t, err)
}

func TestSSMParameter_NukeAll_Error(t *testing.T) {
	t.Parallel()

	mock := &mockSSMParameterClient{
		DeleteParameterError: fmt.Errorf("AccessDenied: not authorized to delete"),
	}
	err := deleteSSMParameter(context.Background(), mock, aws.String("/app/db-password"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "AccessDenied")
}
