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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockEBApplicationsClient struct {
	DescribeApplicationsOutput elasticbeanstalk.DescribeApplicationsOutput
	DeleteApplicationOutput    elasticbeanstalk.DeleteApplicationOutput
}

func (m *mockEBApplicationsClient) DescribeApplications(ctx context.Context, params *elasticbeanstalk.DescribeApplicationsInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DescribeApplicationsOutput, error) {
	return &m.DescribeApplicationsOutput, nil
}

func (m *mockEBApplicationsClient) DeleteApplication(ctx context.Context, params *elasticbeanstalk.DeleteApplicationInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DeleteApplicationOutput, error) {
	return &m.DeleteApplicationOutput, nil
}

func TestListEBApplications(t *testing.T) {
	t.Parallel()

	app1 := "demo-app-golang-backend"
	app2 := "demo-app-golang-frontend"
	now := time.Now()

	mock := &mockEBApplicationsClient{
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
	}

	names, err := listEBApplications(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{app1, app2}, aws.ToStringSlice(names))
}

func TestListEBApplications_WithFilter(t *testing.T) {
	t.Parallel()

	app1 := "demo-app-golang-backend"
	app2 := "skip-this"
	now := time.Now()

	mock := &mockEBApplicationsClient{
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
					DateCreated:     aws.Time(now),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listEBApplications(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{app1}, aws.ToStringSlice(names))
}

func TestListEBApplications_TimeFilter(t *testing.T) {
	t.Parallel()

	app1 := "app1"
	app2 := "app2"
	now := time.Now()

	mock := &mockEBApplicationsClient{
		DescribeApplicationsOutput: elasticbeanstalk.DescribeApplicationsOutput{
			Applications: []types.ApplicationDescription{
				{
					ApplicationName: &app1,
					DateCreated:     aws.Time(now),
				},
				{
					ApplicationName: &app2,
					DateCreated:     aws.Time(now.Add(1)),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now),
		},
	}

	names, err := listEBApplications(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{app1}, aws.ToStringSlice(names))
}

func TestDeleteEBApplication(t *testing.T) {
	t.Parallel()

	mock := &mockEBApplicationsClient{}
	err := deleteEBApplication(context.Background(), mock, aws.String("test-app"))
	require.NoError(t, err)
}
