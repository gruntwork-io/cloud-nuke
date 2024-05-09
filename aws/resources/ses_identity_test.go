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

type mockedSesIdentities struct {
	sesiface.SESAPI
	DeleteIdentityOutput ses.DeleteIdentityOutput
	ListIdentitiesOutput ses.ListIdentitiesOutput
}

func (m mockedSesIdentities) ListIdentitiesWithContext(_ awsgo.Context, _ *ses.ListIdentitiesInput, _ ...request.Option) (*ses.ListIdentitiesOutput, error) {
	return &m.ListIdentitiesOutput, nil
}

func (m mockedSesIdentities) DeleteIdentityWithContext(_ awsgo.Context, _ *ses.DeleteIdentityInput, _ ...request.Option) (*ses.DeleteIdentityOutput, error) {
	return &m.DeleteIdentityOutput, nil
}

func TestSesIdentities_GetAll(t *testing.T) {
	t.Parallel()

	id1 := "test-id-1"
	id2 := "test-id-2"
	identity := SesIdentities{
		Client: mockedSesIdentities{
			ListIdentitiesOutput: ses.ListIdentitiesOutput{
				Identities: []*string{
					awsgo.String(id1),
					awsgo.String(id2),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
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
