package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedSesConfigurationSet struct {
	SesConfigurationSetAPI
	ListConfigurationSetsOutput ses.ListConfigurationSetsOutput
}

func (m mockedSesConfigurationSet) ListConfigurationSets(ctx context.Context, params *ses.ListConfigurationSetsInput, optFns ...func(*ses.Options)) (*ses.ListConfigurationSetsOutput, error) {
	return &m.ListConfigurationSetsOutput, nil
}

func (m mockedSesConfigurationSet) DeleteConfigurationSet(ctx context.Context, params *ses.DeleteConfigurationSetInput, optFns ...func(*ses.Options)) (*ses.DeleteConfigurationSetOutput, error) {
	return &ses.DeleteConfigurationSetOutput{}, nil
}

func TestSesConfigurationSet_GetAll(t *testing.T) {
	t.Parallel()

	mock := mockedSesConfigurationSet{
		ListConfigurationSetsOutput: ses.ListConfigurationSetsOutput{
			ConfigurationSets: []types.ConfigurationSet{
				{Name: aws.String("config-set-1")},
				{Name: aws.String("config-set-2")},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"config-set-1", "config-set-2"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("config-set-1"),
					}},
				},
			},
			expected: []string{"config-set-2"},
		},
		"nameInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("config-set-1"),
					}},
				},
			},
			expected: []string{"config-set-1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listSesConfigurationSets(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSesConfigurationSet_NukeAll(t *testing.T) {
	t.Parallel()

	mock := mockedSesConfigurationSet{}
	err := deleteSesConfigurationSet(context.Background(), mock, aws.String("test-config-set"))
	require.NoError(t, err)
}
