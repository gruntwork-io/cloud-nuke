package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedSesIdentities struct {
	SESIdentityAPI
	DeleteIdentityOutput ses.DeleteIdentityOutput
	ListIdentitiesOutput ses.ListIdentitiesOutput
}

func (m mockedSesIdentities) ListIdentities(_ context.Context, _ *ses.ListIdentitiesInput, _ ...func(*ses.Options)) (*ses.ListIdentitiesOutput, error) {
	return &m.ListIdentitiesOutput, nil
}

func (m mockedSesIdentities) DeleteIdentity(_ context.Context, _ *ses.DeleteIdentityInput, _ ...func(*ses.Options)) (*ses.DeleteIdentityOutput, error) {
	return &m.DeleteIdentityOutput, nil
}

func TestSesIdentities_GetAll(t *testing.T) {
	t.Parallel()

	id1 := "test-id-1"
	id2 := "test-id-2"
	identity := SesIdentities{
		Client: mockedSesIdentities{
			ListIdentitiesOutput: ses.ListIdentitiesOutput{
				Identities: []string{
					(id1),
					(id2),
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
				SESIdentity: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSesIdentities_NukeAll(t *testing.T) {
	t.Parallel()

	identity := SesIdentities{
		Client: mockedSesIdentities{
			DeleteIdentityOutput: ses.DeleteIdentityOutput{},
		},
	}

	err := identity.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
