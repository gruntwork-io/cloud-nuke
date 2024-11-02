package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEBApplication struct {
	EBApplicationsAPI
	DescribeApplicationsOutput elasticbeanstalk.DescribeApplicationsOutput
	DeleteApplicationOutput    elasticbeanstalk.DeleteApplicationOutput
}

func (m *mockedEBApplication) DescribeApplications(ctx context.Context, params *elasticbeanstalk.DescribeApplicationsInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DescribeApplicationsOutput, error) {
	return &m.DescribeApplicationsOutput, nil
}

func (m *mockedEBApplication) DeleteApplication(ctx context.Context, params *elasticbeanstalk.DeleteApplicationInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DeleteApplicationOutput, error) {
	return &m.DeleteApplicationOutput, nil
}

func TestEBApplication_GetAll(t *testing.T) {
	t.Parallel()

	app1 := "demo-app-golang-backend"
	app2 := "demo-app-golang-frontend"

	now := time.Now()
	eb := EBApplications{
		Client: &mockedEBApplication{
			DescribeApplicationsOutput: elasticbeanstalk.DescribeApplicationsOutput{
				Applications: []types.ApplicationDescription{
					{
						ApplicationArn:  aws.String("app-arn-01"),
						ApplicationName: &app1,
						DateCreated:     aws.Time(now),
					},
					{
						ApplicationArn:  aws.String("app-arn-02"),
						ApplicationName: &app2,
						DateCreated:     aws.Time(now.Add(1)),
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
			expected: []string{
				app1, app2,
			},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(app1),
					}}},
			},
			expected: []string{app2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{app1},
		},
		"timeBeforeExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(1)),
				}},
			expected: []string{app2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := eb.getAll(context.Background(), config.Config{
				ElasticBeanstalk: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}
