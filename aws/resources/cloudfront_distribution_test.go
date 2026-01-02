package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

// Compile-time check that the mock implements the interface.
var _ CloudfrontDistributionAPI = (*mockedCloudfrontDistribution)(nil)

type mockedCloudfrontDistribution struct {
	// List operation
	ListDistributionsOutput cloudfront.ListDistributionsOutput
	ListDistributionsErr    error

	// Get operations (used by waiter and for ETag retrieval)
	GetDistributionOutput       cloudfront.GetDistributionOutput
	GetDistributionErr          error
	GetDistributionConfigOutput cloudfront.GetDistributionConfigOutput
	GetDistributionConfigErr    error

	// Update operation
	UpdateDistributionOutput cloudfront.UpdateDistributionOutput
	UpdateDistributionErr    error

	// Delete operation
	DeleteDistributionOutput cloudfront.DeleteDistributionOutput
	DeleteDistributionErr    error
}

func (m *mockedCloudfrontDistribution) ListDistributions(_ context.Context, _ *cloudfront.ListDistributionsInput, _ ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error) {
	return &m.ListDistributionsOutput, m.ListDistributionsErr
}

func (m *mockedCloudfrontDistribution) GetDistribution(_ context.Context, _ *cloudfront.GetDistributionInput, _ ...func(*cloudfront.Options)) (*cloudfront.GetDistributionOutput, error) {
	return &m.GetDistributionOutput, m.GetDistributionErr
}

func (m *mockedCloudfrontDistribution) GetDistributionConfig(_ context.Context, _ *cloudfront.GetDistributionConfigInput, _ ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error) {
	return &m.GetDistributionConfigOutput, m.GetDistributionConfigErr
}

func (m *mockedCloudfrontDistribution) UpdateDistribution(_ context.Context, _ *cloudfront.UpdateDistributionInput, _ ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error) {
	return &m.UpdateDistributionOutput, m.UpdateDistributionErr
}

func (m *mockedCloudfrontDistribution) DeleteDistribution(_ context.Context, _ *cloudfront.DeleteDistributionInput, _ ...func(*cloudfront.Options)) (*cloudfront.DeleteDistributionOutput, error) {
	return &m.DeleteDistributionOutput, m.DeleteDistributionErr
}

func TestCloudfrontDistribution_List(t *testing.T) {
	t.Parallel()

	testID1 := "E1EXAMPLE1"
	testID2 := "E2EXAMPLE2"

	tests := map[string]struct {
		listOutput cloudfront.ListDistributionsOutput
		configObj  config.ResourceType
		expected   []string
	}{
		"returns all distributions when no filter": {
			listOutput: cloudfront.ListDistributionsOutput{
				DistributionList: &types.DistributionList{
					Items: []types.DistributionSummary{
						{Id: aws.String(testID1)},
						{Id: aws.String(testID2)},
					},
				},
			},
			configObj: config.ResourceType{},
			expected:  []string{testID1, testID2},
		},
		"filters distributions by exclusion pattern": {
			listOutput: cloudfront.ListDistributionsOutput{
				DistributionList: &types.DistributionList{
					Items: []types.DistributionSummary{
						{Id: aws.String(testID1)},
						{Id: aws.String(testID2)},
					},
				},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("E1.*")}},
				},
			},
			expected: []string{testID2},
		},
		"handles empty distribution list": {
			listOutput: cloudfront.ListDistributionsOutput{
				DistributionList: &types.DistributionList{},
			},
			configObj: config.ResourceType{},
			expected:  []string{},
		},
		"handles nil distribution list": {
			listOutput: cloudfront.ListDistributionsOutput{},
			configObj:  config.ResourceType{},
			expected:   []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &mockedCloudfrontDistribution{
				ListDistributionsOutput: tc.listOutput,
			}

			ids, err := listCloudfrontDistributions(context.Background(), client, resource.Scope{}, tc.configObj)

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestCloudfrontDistribution_Nuke_AlreadyDisabled(t *testing.T) {
	t.Parallel()

	testID := aws.String("E1EXAMPLE1")
	testETag := aws.String("ETAG123")

	client := &mockedCloudfrontDistribution{
		GetDistributionConfigOutput: cloudfront.GetDistributionConfigOutput{
			ETag: testETag,
			DistributionConfig: &types.DistributionConfig{
				Enabled: aws.Bool(false), // Already disabled
			},
		},
		DeleteDistributionOutput: cloudfront.DeleteDistributionOutput{},
	}

	err := nukeCloudfrontDistribution(context.Background(), client, testID)
	require.NoError(t, err)
}

func TestCloudfrontDistribution_Nuke_EnabledWithDeployedState(t *testing.T) {
	t.Parallel()

	testID := aws.String("E1EXAMPLE1")
	testETag := aws.String("ETAG123")
	updatedETag := aws.String("ETAG456")

	callCount := 0
	client := &mockedCloudfrontDistribution{
		GetDistributionConfigOutput: cloudfront.GetDistributionConfigOutput{
			ETag: testETag,
			DistributionConfig: &types.DistributionConfig{
				Enabled: aws.Bool(true), // Currently enabled
			},
		},
		UpdateDistributionOutput: cloudfront.UpdateDistributionOutput{
			ETag: updatedETag,
		},
		// GetDistribution is called by the waiter - return Deployed status
		GetDistributionOutput: cloudfront.GetDistributionOutput{
			ETag: updatedETag,
			Distribution: &types.Distribution{
				Status: aws.String("Deployed"),
			},
		},
		DeleteDistributionOutput: cloudfront.DeleteDistributionOutput{},
	}

	// Override GetDistributionConfig to track calls and return different values
	originalGetDistributionConfig := client.GetDistributionConfigOutput
	client.GetDistributionConfigOutput = originalGetDistributionConfig

	// Create a custom mock that changes behavior after update
	customClient := &dynamicMockedCloudfrontDistribution{
		mockedCloudfrontDistribution: client,
		getDistributionConfigCalls:   &callCount,
		initialETag:                  testETag,
		updatedETag:                  updatedETag,
	}

	err := nukeCloudfrontDistribution(context.Background(), customClient, testID)
	require.NoError(t, err)
}

// dynamicMockedCloudfrontDistribution extends the mock to handle multiple GetDistributionConfig calls
type dynamicMockedCloudfrontDistribution struct {
	*mockedCloudfrontDistribution
	getDistributionConfigCalls *int
	initialETag                *string
	updatedETag                *string
}

func (m *dynamicMockedCloudfrontDistribution) GetDistributionConfig(_ context.Context, _ *cloudfront.GetDistributionConfigInput, _ ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error) {
	*m.getDistributionConfigCalls++

	// First call returns enabled=true, subsequent calls return enabled=false (after update)
	if *m.getDistributionConfigCalls == 1 {
		return &cloudfront.GetDistributionConfigOutput{
			ETag: m.initialETag,
			DistributionConfig: &types.DistributionConfig{
				Enabled: aws.Bool(true),
			},
		}, nil
	}

	return &cloudfront.GetDistributionConfigOutput{
		ETag: m.updatedETag,
		DistributionConfig: &types.DistributionConfig{
			Enabled: aws.Bool(false),
		},
	}, nil
}
