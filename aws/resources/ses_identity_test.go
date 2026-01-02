package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

// mockSESIdentityClient implements SESIdentityAPI for testing.
type mockSESIdentityClient struct {
	ListIdentitiesOutput ses.ListIdentitiesOutput
	DeleteIdentityOutput ses.DeleteIdentityOutput
	DeleteIdentityError  error
}

func (m *mockSESIdentityClient) ListIdentities(_ context.Context, _ *ses.ListIdentitiesInput, _ ...func(*ses.Options)) (*ses.ListIdentitiesOutput, error) {
	return &m.ListIdentitiesOutput, nil
}

func (m *mockSESIdentityClient) DeleteIdentity(_ context.Context, _ *ses.DeleteIdentityInput, _ ...func(*ses.Options)) (*ses.DeleteIdentityOutput, error) {
	return &m.DeleteIdentityOutput, m.DeleteIdentityError
}

func TestSesIdentities_GetAll(t *testing.T) {
	t.Parallel()

	id1 := "test-identity-1@example.com"
	id2 := "test-identity-2@example.com"

	mock := &mockSESIdentityClient{
		ListIdentitiesOutput: ses.ListIdentitiesOutput{
			Identities: []string{id1, id2},
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
						RE: *regexp.MustCompile("test-identity-2"),
					}},
				},
			},
			expected: []string{id1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listSesIdentities(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestSesIdentities_Delete(t *testing.T) {
	t.Parallel()

	mock := &mockSESIdentityClient{
		DeleteIdentityOutput: ses.DeleteIdentityOutput{},
	}

	err := deleteSesIdentity(context.Background(), mock, aws.String("test@example.com"))
	require.NoError(t, err)
}
