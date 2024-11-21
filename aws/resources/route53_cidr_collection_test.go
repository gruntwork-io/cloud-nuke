package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedR53CidrCollection struct {
	Route53CidrCollectionAPI
	ListCidrCollectionsOutput  route53.ListCidrCollectionsOutput
	ListCidrBlocksOutput       route53.ListCidrBlocksOutput
	ChangeCidrCollectionOutput route53.ChangeCidrCollectionOutput
	DeleteCidrCollectionOutput route53.DeleteCidrCollectionOutput
}

func (mock mockedR53CidrCollection) ListCidrBlocks(_ context.Context, _ *route53.ListCidrBlocksInput, _ ...func(*route53.Options)) (*route53.ListCidrBlocksOutput, error) {
	return &mock.ListCidrBlocksOutput, nil
}

func (mock mockedR53CidrCollection) ChangeCidrCollection(_ context.Context, _ *route53.ChangeCidrCollectionInput, _ ...func(*route53.Options)) (*route53.ChangeCidrCollectionOutput, error) {
	return &mock.ChangeCidrCollectionOutput, nil
}

func (mock mockedR53CidrCollection) ListCidrCollections(_ context.Context, _ *route53.ListCidrCollectionsInput, _ ...func(*route53.Options)) (*route53.ListCidrCollectionsOutput, error) {
	return &mock.ListCidrCollectionsOutput, nil
}

func (mock mockedR53CidrCollection) DeleteCidrCollection(_ context.Context, _ *route53.DeleteCidrCollectionInput, _ ...func(*route53.Options)) (*route53.DeleteCidrCollectionOutput, error) {
	return &mock.DeleteCidrCollectionOutput, nil
}

func TestR53CidrCollection_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "d8c6f2db-89dd-5533-f30c-13e28eba8818"
	testId2 := "d8c6f2db-90dd-5533-f30c-13e28eba8818"

	testName1 := "Test name 01"
	testName2 := "Test name 02"
	rc := Route53CidrCollection{
		Client: mockedR53CidrCollection{
			ListCidrCollectionsOutput: route53.ListCidrCollectionsOutput{
				CidrCollections: []types.CollectionSummary{
					{
						Id:   aws.String(testId1),
						Name: aws.String(testName1),
					},
					{
						Id:   aws.String(testId2),
						Name: aws.String(testName2),
					},
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
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := rc.getAll(context.Background(), config.Config{
				Route53CIDRCollection: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestR53CidrCollection_Nuke(t *testing.T) {

	t.Parallel()

	rc := Route53CidrCollection{
		Client: mockedR53CidrCollection{
			ListCidrBlocksOutput: route53.ListCidrBlocksOutput{
				CidrBlocks: []types.CidrBlockSummary{
					{
						CidrBlock:    aws.String("222::0"),
						LocationName: aws.String("sample-location-01"),
					},
				},
			},
			ChangeCidrCollectionOutput: route53.ChangeCidrCollectionOutput{},
			ListCidrCollectionsOutput: route53.ListCidrCollectionsOutput{
				CidrCollections: []types.CollectionSummary{
					{
						Id:   aws.String("collection-id-01"),
						Name: aws.String("collection-name-01"),
					},
				},
			},
			DeleteCidrCollectionOutput: route53.DeleteCidrCollectionOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("collection-id-01")})
	require.NoError(t, err)
}
