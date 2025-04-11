package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

var _ CloudfrontDistributionAPI = mockedCloudfrontDistribution{}

type mockedCloudfrontDistribution struct {
	ListDistributionsOutput     cloudfront.ListDistributionsOutput
	GetDistributionConfigOutput cloudfront.GetDistributionConfigOutput
	UpdateDistributionOutput    cloudfront.UpdateDistributionOutput
	DeleteDistributionOutput    cloudfront.DeleteDistributionOutput
}

func (m mockedCloudfrontDistribution) ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error) {
	return &m.ListDistributionsOutput, nil
}
func (m mockedCloudfrontDistribution) GetDistributionConfig(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error) {
	return &m.GetDistributionConfigOutput, nil
}
func (m mockedCloudfrontDistribution) UpdateDistribution(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error) {
	return &m.UpdateDistributionOutput, nil
}
func (m mockedCloudfrontDistribution) DeleteDistribution(ctx context.Context, params *cloudfront.DeleteDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.DeleteDistributionOutput, error) {
	return &m.DeleteDistributionOutput, nil
}

func TestCloudfrontDistributionGetAll(t *testing.T) {
	t.Parallel()

	testId1 := "test-id1"
	testId2 := "test-id2"
	cd := CloudfrontDistribution{
		Client: mockedCloudfrontDistribution{
			ListDistributionsOutput: cloudfront.ListDistributionsOutput{
				DistributionList: &types.DistributionList{
					Items: []types.DistributionSummary{
						{
							Id: aws.String(testId1),
						},
						{
							Id: aws.String(testId2),
						},
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
						RE: *regexp.MustCompile(testId1),
					}}},
			},
			expected: []string{testId2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := cd.getAll(context.Background(), config.Config{
				CloudfrontDistribution: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestCloudfrontDistributionNukeAll(t *testing.T) {
	t.Parallel()

	client := &mockedCloudfrontDistribution{
		GetDistributionConfigOutput: cloudfront.GetDistributionConfigOutput{
			DistributionConfig: &types.DistributionConfig{
				Enabled: aws.Bool(true),
			},
			ETag: aws.String("etag"),
		},
		UpdateDistributionOutput: cloudfront.UpdateDistributionOutput{},
		DeleteDistributionOutput: cloudfront.DeleteDistributionOutput{},
	}
	cd := CloudfrontDistribution{
		Client: client,
	}

	time.AfterFunc(time.Second, func() {
		client.GetDistributionConfigOutput = cloudfront.GetDistributionConfigOutput{
			DistributionConfig: &types.DistributionConfig{
				Enabled: aws.Bool(false),
			},
			ETag: aws.String("etag"),
		}
	})

	err := cd.nukeAll([]*string{aws.String("test-arn")})
	require.NoError(t, err)
}
