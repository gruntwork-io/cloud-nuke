package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedRoute53CidrCollectionAPI struct {
	Route53CidrCollectionAPI
	ListCidrCollectionsOutput  route53.ListCidrCollectionsOutput
	ListCidrBlocksOutput       route53.ListCidrBlocksOutput
	ChangeCidrCollectionOutput route53.ChangeCidrCollectionOutput
	DeleteCidrCollectionOutput route53.DeleteCidrCollectionOutput
}

func (m mockedRoute53CidrCollectionAPI) ListCidrCollections(_ context.Context, _ *route53.ListCidrCollectionsInput, _ ...func(*route53.Options)) (*route53.ListCidrCollectionsOutput, error) {
	return &m.ListCidrCollectionsOutput, nil
}

func (m mockedRoute53CidrCollectionAPI) ListCidrBlocks(_ context.Context, _ *route53.ListCidrBlocksInput, _ ...func(*route53.Options)) (*route53.ListCidrBlocksOutput, error) {
	return &m.ListCidrBlocksOutput, nil
}

func (m mockedRoute53CidrCollectionAPI) ChangeCidrCollection(_ context.Context, _ *route53.ChangeCidrCollectionInput, _ ...func(*route53.Options)) (*route53.ChangeCidrCollectionOutput, error) {
	return &m.ChangeCidrCollectionOutput, nil
}

func (m mockedRoute53CidrCollectionAPI) DeleteCidrCollection(_ context.Context, _ *route53.DeleteCidrCollectionInput, _ ...func(*route53.Options)) (*route53.DeleteCidrCollectionOutput, error) {
	return &m.DeleteCidrCollectionOutput, nil
}

func TestListRoute53CidrCollections(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		collections []types.CollectionSummary
		configObj   config.ResourceType
		expected    []string
	}{
		"emptyFilter": {
			collections: []types.CollectionSummary{
				{Id: aws.String("id-1"), Name: aws.String("collection-1")},
				{Id: aws.String("id-2"), Name: aws.String("collection-2")},
			},
			configObj: config.ResourceType{},
			expected:  []string{"id-1", "id-2"},
		},
		"nameExclusionFilter": {
			collections: []types.CollectionSummary{
				{Id: aws.String("id-1"), Name: aws.String("collection-1")},
				{Id: aws.String("id-2"), Name: aws.String("collection-2")},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("collection-1")}},
				},
			},
			expected: []string{"id-2"},
		},
		"emptyCollections": {
			collections: []types.CollectionSummary{},
			configObj:   config.ResourceType{},
			expected:    []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := mockedRoute53CidrCollectionAPI{
				ListCidrCollectionsOutput: route53.ListCidrCollectionsOutput{
					CidrCollections: tc.collections,
				},
			}

			ids, err := listRoute53CidrCollections(context.Background(), client, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestNukeCidrBlocks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		blocks    []types.CidrBlockSummary
		expectErr bool
	}{
		"withBlocks": {
			blocks: []types.CidrBlockSummary{
				{CidrBlock: aws.String("10.0.0.0/8"), LocationName: aws.String("location-1")},
				{CidrBlock: aws.String("192.168.0.0/16"), LocationName: aws.String("location-2")},
			},
			expectErr: false,
		},
		"noBlocks": {
			blocks:    []types.CidrBlockSummary{},
			expectErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := mockedRoute53CidrCollectionAPI{
				ListCidrBlocksOutput:       route53.ListCidrBlocksOutput{CidrBlocks: tc.blocks},
				ChangeCidrCollectionOutput: route53.ChangeCidrCollectionOutput{},
			}

			err := nukeCidrBlocks(context.Background(), client, aws.String("test-id"))
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteRoute53CidrCollection(t *testing.T) {
	t.Parallel()

	client := mockedRoute53CidrCollectionAPI{
		DeleteCidrCollectionOutput: route53.DeleteCidrCollectionOutput{},
	}

	err := deleteRoute53CidrCollection(context.Background(), client, aws.String("test-id"))
	require.NoError(t, err)
}
