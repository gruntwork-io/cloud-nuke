package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCloudWatchDashboards(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	cwdbName := createCloudWatchDashboard(t, svc, region)
	defer deleteCloudWatchDashboard(t, svc, cwdbName, true)

	cwdbNames, err := getAllCloudWatchDashboards(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(cwdbNames), aws.StringValue(cwdbName))
}

func TestTimeFilterExclusionNewlyCreatedCloudWatchDashboard(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	cwdbName := createCloudWatchDashboard(t, svc, region)
	defer deleteCloudWatchDashboard(t, svc, cwdbName, true)

	// Assert CloudWatch Dashboard is picked up without filters
	cwdbNamesNewer, err := getAllCloudWatchDashboards(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(cwdbNamesNewer), aws.StringValue(cwdbName))

	// Assert user doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	cwdbNamesOlder, err := getAllCloudWatchDashboards(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(cwdbNamesOlder), aws.StringValue(cwdbName))
}

func TestNukeCloudWatchDashboardOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	cwdbName := createCloudWatchDashboard(t, svc, region)
	defer deleteCloudWatchDashboard(t, svc, cwdbName, false)
	identifiers := []*string{cwdbName}

	require.NoError(
		t,
		nukeAllCloudWatchDashboards(session, identifiers),
	)

	// Make sure the CloudWatch Dashboard is deleted.
	assertCloudWatchDashboardsDeleted(t, svc, identifiers)
}

func TestNukeCloudWatchDashboardsMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	cwdbNames := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		cwdbName := createCloudWatchDashboard(t, svc, region)
		defer deleteCloudWatchDashboard(t, svc, cwdbName, false)
		cwdbNames = append(cwdbNames, cwdbName)
	}

	require.NoError(
		t,
		nukeAllCloudWatchDashboards(session, cwdbNames),
	)

	// Make sure the CloudWatch Dashboard is deleted.
	assertCloudWatchDashboardsDeleted(t, svc, cwdbNames)
}

// Helper functions for driving the CloudWatch Dashboard tests

// createCloudWatchDashboard will create a new CloudWatch Dashboard with a single text widget.
func createCloudWatchDashboard(t *testing.T, svc *cloudwatch.CloudWatch, region string) *string {
	uniqueID := util.UniqueID()
	name := fmt.Sprintf("cloud-nuke-test-%s", strings.ToLower(uniqueID))

	resp, err := svc.PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardBody: aws.String(helloWorldCloudWatchDashboardWidget),
		DashboardName: aws.String(name),
	})
	require.NoError(t, err)
	if len(resp.DashboardValidationMessages) > 0 {
		t.Fatalf("Error creating Dashboard %v", resp.DashboardValidationMessages)
	}
	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

// deleteCloudWatchDashboard is a function to delete the given CloudWatch Dashboard.
func deleteCloudWatchDashboard(t *testing.T, svc *cloudwatch.CloudWatch, name *string, checkErr bool) {
	input := &cloudwatch.DeleteDashboardsInput{DashboardNames: []*string{name}}
	_, err := svc.DeleteDashboards(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func assertCloudWatchDashboardsDeleted(t *testing.T, svc *cloudwatch.CloudWatch, identifiers []*string) {
	for _, name := range identifiers {
		resp, err := svc.ListDashboards(&cloudwatch.ListDashboardsInput{DashboardNamePrefix: name})
		require.NoError(t, err)
		if len(resp.DashboardEntries) > 0 {
			t.Fatalf("Dashboard %s is not deleted", aws.StringValue(name))
		}
	}
}

const helloWorldCloudWatchDashboardWidget = `{
   "widgets":[
      {
         "type":"text",
         "x":0,
         "y":7,
         "width":3,
         "height":3,
         "properties":{
            "markdown":"Hello world"
         }
      }
   ]
}`
