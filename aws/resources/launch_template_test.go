package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockLaunchTemplatesClient struct {
	DescribeLaunchTemplatesOutput ec2.DescribeLaunchTemplatesOutput
	DeleteLaunchTemplateOutput    ec2.DeleteLaunchTemplateOutput
}

func (m *mockLaunchTemplatesClient) DescribeLaunchTemplates(ctx context.Context, params *ec2.DescribeLaunchTemplatesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeLaunchTemplatesOutput, error) {
	return &m.DescribeLaunchTemplatesOutput, nil
}

func (m *mockLaunchTemplatesClient) DeleteLaunchTemplate(ctx context.Context, params *ec2.DeleteLaunchTemplateInput, optFns ...func(*ec2.Options)) (*ec2.DeleteLaunchTemplateOutput, error) {
	return &m.DeleteLaunchTemplateOutput, nil
}

func TestListLaunchTemplates(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockLaunchTemplatesClient{
		DescribeLaunchTemplatesOutput: ec2.DescribeLaunchTemplatesOutput{
			LaunchTemplates: []types.LaunchTemplate{
				{
					LaunchTemplateName: aws.String("template1"),
					LaunchTemplateId:   aws.String("lt-123"),
					CreateTime:         aws.Time(now),
				},
				{
					LaunchTemplateName: aws.String("template2"),
					LaunchTemplateId:   aws.String("lt-456"),
					CreateTime:         aws.Time(now.Add(time.Hour)),
					Tags: []types.Tag{
						{Key: aws.String("Environment"), Value: aws.String("test")},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"noFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"template1", "template2"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("template1")}},
				},
			},
			expected: []string{"template2"},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{"template1"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listLaunchTemplates(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteLaunchTemplate(t *testing.T) {
	t.Parallel()

	mock := &mockLaunchTemplatesClient{}
	err := deleteLaunchTemplate(context.Background(), mock, aws.String("test-template"))
	require.NoError(t, err)
}
