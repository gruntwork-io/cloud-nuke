package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCloudWatchLogGroups(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := cloudwatchlogs.New(session)

	lgName := createCloudWatchLogGroup(t, svc, region)
	defer deleteCloudWatchLogGroup(t, svc, lgName, true)

	lgNames, err := getAllCloudWatchLogGroups(session, region)
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(lgNames), aws.StringValue(lgName))
}

func TestNukeCloudWatchLogGroupOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatchlogs.New(session)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	lgName := createCloudWatchLogGroup(t, svc, region)
	defer deleteCloudWatchLogGroup(t, svc, lgName, false)
	identifiers := []*string{lgName}

	require.NoError(
		t,
		nukeAllCloudWatchLogGroups(session, identifiers),
	)

	// Make sure the CloudWatch Dashboard is deleted.
	assertCloudWatchLogGroupsDeleted(t, svc, identifiers)
}

func TestNukeCloudWatchLogGroupMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatchlogs.New(session)

	lgNames := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		lgName := createCloudWatchLogGroup(t, svc, region)
		defer deleteCloudWatchLogGroup(t, svc, lgName, false)
		lgNames = append(lgNames, lgName)
	}

	require.NoError(
		t,
		nukeAllCloudWatchLogGroups(session, lgNames),
	)

	// Make sure the CloudWatch Dashboard is deleted.
	assertCloudWatchLogGroupsDeleted(t, svc, lgNames)
}

func createCloudWatchLogGroup(t *testing.T, svc *cloudwatchlogs.CloudWatchLogs, region string) *string {
	uniqueID := util.UniqueID()
	name := fmt.Sprintf("cloud-nuke-test-%s", strings.ToLower(uniqueID))

	_, err := svc.CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(name),
	})
	require.NoError(t, err)

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

// deleteCloudWatchLogGroup is a function to delete the given CloudWatch Log Group.
func deleteCloudWatchLogGroup(t *testing.T, svc *cloudwatchlogs.CloudWatchLogs, name *string, checkErr bool) {
	_, err := svc.DeleteLogGroup(&cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: name,
	})
	if checkErr {
		require.NoError(t, err)
	}
}

func assertCloudWatchLogGroupsDeleted(t *testing.T, svc *cloudwatchlogs.CloudWatchLogs, identifiers []*string) {
	for _, name := range identifiers {
		resp, err := svc.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{
			LogGroupNamePrefix: name,
		})
		require.NoError(t, err)
		if len(resp.LogGroups) > 0 {
			t.Fatalf("Log Group %s is not deleted", aws.StringValue(name))
		}
	}
}
