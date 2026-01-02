package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockSesEmailTemplatesClient struct {
	ListTemplatesOutput  ses.ListTemplatesOutput
	DeleteTemplateOutput ses.DeleteTemplateOutput
}

func (m *mockSesEmailTemplatesClient) ListTemplates(ctx context.Context, params *ses.ListTemplatesInput, optFns ...func(*ses.Options)) (*ses.ListTemplatesOutput, error) {
	return &m.ListTemplatesOutput, nil
}

func (m *mockSesEmailTemplatesClient) DeleteTemplate(ctx context.Context, params *ses.DeleteTemplateInput, optFns ...func(*ses.Options)) (*ses.DeleteTemplateOutput, error) {
	return &m.DeleteTemplateOutput, nil
}

func TestListSesEmailTemplates(t *testing.T) {
	t.Parallel()

	testName1 := "template1"
	testName2 := "template2"
	now := time.Now()

	mock := &mockSesEmailTemplatesClient{
		ListTemplatesOutput: ses.ListTemplatesOutput{
			TemplatesMetadata: []types.TemplateMetadata{
				{Name: aws.String(testName1), CreatedTimestamp: aws.Time(now)},
				{Name: aws.String(testName2), CreatedTimestamp: aws.Time(now.Add(1 * time.Hour))},
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
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listSesEmailTemplates(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSesEmailTemplate(t *testing.T) {
	t.Parallel()

	mock := &mockSesEmailTemplatesClient{}
	err := deleteSesEmailTemplate(context.Background(), mock, aws.String("test-template"))
	require.NoError(t, err)
}
