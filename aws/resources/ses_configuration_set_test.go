package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedSesConfigurationSet struct {
	sesiface.SESAPI
	DeleteConfigurationSetOutput ses.DeleteConfigurationSetOutput
	ListConfigurationSetsOutput  ses.ListConfigurationSetsOutput
}

func (m mockedSesConfigurationSet) ListConfigurationSetsWithContext(_ awsgo.Context, _ *ses.ListConfigurationSetsInput, _ ...request.Option) (*ses.ListConfigurationSetsOutput, error) {
	return &m.ListConfigurationSetsOutput, nil
}

func (m mockedSesConfigurationSet) DeleteConfigurationSetWithContext(_ awsgo.Context, _ *ses.DeleteConfigurationSetInput, _ ...request.Option) (*ses.DeleteConfigurationSetOutput, error) {
	return &m.DeleteConfigurationSetOutput, nil
}

var (
	id1                = "test-id-1"
	id2                = "test-id-2"
	configurationsSet1 = ses.ConfigurationSet{
		Name: awsgo.String(id1),
	}
	configurationsSet2 = ses.ConfigurationSet{
		Name: awsgo.String(id2),
	}
)

func TestSesConfigurationSet_GetAll(t *testing.T) {
	t.Parallel()

	identity := SesConfigurationSet{
		Client: mockedSesConfigurationSet{
			ListConfigurationSetsOutput: ses.ListConfigurationSetsOutput{
				ConfigurationSets: []*ses.ConfigurationSet{
					&configurationsSet1,
					&configurationsSet2,
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := identity.getAll(context.Background(), config.Config{
				SESConfigurationSet: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestSesConfigurationSet_NukeAll(t *testing.T) {
	t.Parallel()

	identity := SesConfigurationSet{
		Client: mockedSesConfigurationSet{},
	}

	err := identity.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
