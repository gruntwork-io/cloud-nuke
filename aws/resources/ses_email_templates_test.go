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
	"github.com/stretchr/testify/assert"
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

func TestSesEmailTemplates_ResourceName(t *testing.T) {
	r := NewSesEmailTemplates()
	assert.Equal(t, "ses-email-template", r.ResourceName())
}

func TestSesEmailTemplates_MaxBatchSize(t *testing.T) {
	r := NewSesEmailTemplates()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListSesEmailTemplates(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockSesEmailTemplatesClient{
		ListTemplatesOutput: ses.ListTemplatesOutput{
			TemplatesMetadata: []types.TemplateMetadata{
				{Name: aws.String("template1"), CreatedTimestamp: aws.Time(now)},
				{Name: aws.String("template2"), CreatedTimestamp: aws.Time(now)},
			},
		},
	}

	names, err := listSesEmailTemplates(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"template1", "template2"}, aws.ToStringSlice(names))
}

func TestListSesEmailTemplates_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockSesEmailTemplatesClient{
		ListTemplatesOutput: ses.ListTemplatesOutput{
			TemplatesMetadata: []types.TemplateMetadata{
				{Name: aws.String("template1"), CreatedTimestamp: aws.Time(now)},
				{Name: aws.String("skip-this"), CreatedTimestamp: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listSesEmailTemplates(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"template1"}, aws.ToStringSlice(names))
}

func TestDeleteSesEmailTemplate(t *testing.T) {
	t.Parallel()

	mock := &mockSesEmailTemplatesClient{}
	err := deleteSesEmailTemplate(context.Background(), mock, aws.String("test-template"))
	require.NoError(t, err)
}
