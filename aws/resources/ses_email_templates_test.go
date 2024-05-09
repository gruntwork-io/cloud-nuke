package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedSesEmailTemplates struct {
	sesiface.SESAPI
	DeleteTemplateOutput ses.DeleteTemplateOutput
	ListTemplatesOutput  ses.ListTemplatesOutput
}

func (m mockedSesEmailTemplates) ListTemplatesWithContext(_ awsgo.Context, _ *ses.ListTemplatesInput, _ ...request.Option) (*ses.ListTemplatesOutput, error) {
	return &m.ListTemplatesOutput, nil
}

func (m mockedSesEmailTemplates) DeleteTemplateWithContext(_ awsgo.Context, _ *ses.DeleteTemplateInput, _ ...request.Option) (*ses.DeleteTemplateOutput, error) {
	return &m.DeleteTemplateOutput, nil
}

func TestSesEmailTemplates_GetAll(t *testing.T) {

	id1 := "test-id-1"
	id2 := "test-id-2"
	templateMetadata1 := ses.TemplateMetadata{
		CreatedTimestamp: aws.Time(time.Now()),
		Name:             aws.String(id1),
	}
	templateMetadata2 := ses.TemplateMetadata{
		CreatedTimestamp: aws.Time(time.Now().AddDate(-1, 0, 0)),
		Name:             aws.String(id2),
	}
	t.Parallel()

	sesEmailTemp := SesEmailTemplates{
		Client: mockedSesEmailTemplates{
			ListTemplatesOutput: ses.ListTemplatesOutput{
				TemplatesMetadata: []*ses.TemplateMetadata{
					&templateMetadata1,
					&templateMetadata2,
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
			expected:  []string{id1, id2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(id2),
					}}},
			},
			expected: []string{id1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(time.Now().Add(-1 * time.Hour)),
				}},
			expected: []string{id2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := sesEmailTemp.getAll(context.Background(), config.Config{
				SESEmailTemplates: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestSesEmailTemplates_NukeAll(t *testing.T) {
	t.Parallel()

	sesEmailTemp := SesEmailTemplates{
		Client: mockedSesEmailTemplates{},
	}

	err := sesEmailTemp.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
