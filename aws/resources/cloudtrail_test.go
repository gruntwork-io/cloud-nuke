package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCloudTrail struct {
	CloudtrailTrailAPI
	ListTrailsOutput  cloudtrail.ListTrailsOutput
	DeleteTrailOutput cloudtrail.DeleteTrailOutput
}

func (m mockedCloudTrail) ListTrails(ctx context.Context, params *cloudtrail.ListTrailsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.ListTrailsOutput, error) {
	return &m.ListTrailsOutput, nil
}

func (m mockedCloudTrail) DeleteTrail(ctx context.Context, params *cloudtrail.DeleteTrailInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.DeleteTrailOutput, error) {
	return &m.DeleteTrailOutput, nil
}

func TestCloudTrailGetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testArn1 := "test-arn1"
	testArn2 := "test-arn2"
	ct := CloudtrailTrail{
		Client: mockedCloudTrail{
			ListTrailsOutput: cloudtrail.ListTrailsOutput{
				Trails: []types.TrailInfo{
					{
						Name:     aws.String(testName1),
						TrailARN: aws.String(testArn1),
					},
					{
						Name:     aws.String(testName2),
						TrailARN: aws.String(testArn2),
					},
				}}}}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ct.getAll(context.Background(), config.Config{
				CloudtrailTrail: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestCloudTrailNukeAll(t *testing.T) {
	t.Parallel()

	ct := CloudtrailTrail{
		Client: mockedCloudTrail{
			DeleteTrailOutput: cloudtrail.DeleteTrailOutput{},
		}}

	err := ct.nukeAll([]*string{aws.String("test-arn")})
	require.NoError(t, err)
}
