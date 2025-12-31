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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSesConfigurationSetClient struct {
	ListConfigurationSetsOutput  ses.ListConfigurationSetsOutput
	DeleteConfigurationSetOutput ses.DeleteConfigurationSetOutput
}

func (m *mockSesConfigurationSetClient) ListConfigurationSets(ctx context.Context, params *ses.ListConfigurationSetsInput, optFns ...func(*ses.Options)) (*ses.ListConfigurationSetsOutput, error) {
	return &m.ListConfigurationSetsOutput, nil
}

func (m *mockSesConfigurationSetClient) DeleteConfigurationSet(ctx context.Context, params *ses.DeleteConfigurationSetInput, optFns ...func(*ses.Options)) (*ses.DeleteConfigurationSetOutput, error) {
	return &m.DeleteConfigurationSetOutput, nil
}

func TestSesConfigurationSet_ResourceName(t *testing.T) {
	r := NewSesConfigurationSet()
	assert.Equal(t, "ses-configuration-set", r.ResourceName())
}

func TestSesConfigurationSet_MaxBatchSize(t *testing.T) {
	r := NewSesConfigurationSet()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListSesConfigurationSets(t *testing.T) {
	t.Parallel()

	mock := &mockSesConfigurationSetClient{
		ListConfigurationSetsOutput: ses.ListConfigurationSetsOutput{
			ConfigurationSets: []types.ConfigurationSet{
				{Name: aws.String("config-set-1")},
				{Name: aws.String("config-set-2")},
			},
		},
	}

	names, err := listSesConfigurationSets(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"config-set-1", "config-set-2"}, aws.ToStringSlice(names))
}

func TestListSesConfigurationSets_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockSesConfigurationSetClient{
		ListConfigurationSetsOutput: ses.ListConfigurationSetsOutput{
			ConfigurationSets: []types.ConfigurationSet{
				{Name: aws.String("config-set-1")},
				{Name: aws.String("skip-this")},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listSesConfigurationSets(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"config-set-1"}, aws.ToStringSlice(names))
}

func TestDeleteSesConfigurationSet(t *testing.T) {
	t.Parallel()

	mock := &mockSesConfigurationSetClient{}
	err := deleteSesConfigurationSet(context.Background(), mock, aws.String("test-config-set"))
	require.NoError(t, err)
}
