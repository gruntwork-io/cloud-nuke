package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
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

func TestListCloudWatchAlarms(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	// cwal : CloudWatch ALarms (recommended by ChatGPT)
	cwalName := createCloudWatchAlarm(t, svc, region)
	defer deleteCloudWatchAlarm(t, svc, cwalName, true)

	cwalNames, err := getAllCloudWatchAlarms(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(cwalNames), aws.StringValue(cwalName))
}

func TestTimeFilterExclusionNewlyCreatedCloudWatchAlarm(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	cwalName := createCloudWatchAlarm(t, svc, region)
	defer deleteCloudWatchAlarm(t, svc, cwalName, true)

	// Assert CloudWatch Alarm is picked up without filters
	cwalNamesNewer, err := getAllCloudWatchAlarms(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(cwalNamesNewer), aws.StringValue(cwalName))

	// Assert user doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	cwalNamesOlder, err := getAllCloudWatchAlarms(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(cwalNamesOlder), aws.StringValue(cwalName))
}

func TestNukeCloudWatchAlarmOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	cwalName := createCloudWatchAlarm(t, svc, region)
	defer deleteCloudWatchAlarm(t, svc, cwalName, false)
	identifiers := []*string{cwalName}

	require.NoError(
		t,
		nukeAllCloudWatchAlarms(session, identifiers),
	)

	// Make sure the CloudWatch Alar m is deleted.
	assertCloudWatchAlarmsDeleted(t, svc, identifiers)
}

func TestNukeCloudWatchAlarmsMoreThanOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := cloudwatch.New(session)

	cwalNames := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		cwalName := createCloudWatchAlarm(t, svc, region)
		defer deleteCloudWatchAlarm(t, svc, cwalName, false)
		cwalNames = append(cwalNames, cwalName)
	}

	require.NoError(
		t,
		nukeAllCloudWatchAlarms(session, cwalNames),
	)

	// Make sure the CloudWatch Alarm is deleted.
	assertCloudWatchAlarmsDeleted(t, svc, cwalNames)
}

// Helper functions for driving the CloudWatch Alarm tests

// createCloudWatchAlarm will create a new CloudWatch Alarm with a simple metric.
func createCloudWatchAlarm(t *testing.T, svc *cloudwatch.CloudWatch, region string) *string {
	uniqueID := util.UniqueID()
	name := fmt.Sprintf("cloud-nuke-testing-%s", strings.ToLower(uniqueID))
	metric := []*cloudwatch.MetricDataQuery{}
	metric = append(metric, &cloudwatch.MetricDataQuery{})

	_, err := svc.PutMetricAlarm(&cloudwatch.PutMetricAlarmInput{
		ActionsEnabled:     aws.Bool(true),
		AlarmActions:       aws.StringSlice([]string{}),
		AlarmDescription:   aws.String(`Test alarm for cloud-nuke.`),
		AlarmName:          aws.String(name),
		ComparisonOperator: aws.String(`GreaterThanThreshold`),
		DatapointsToAlarm:  aws.Int64(1),
		Dimensions: []*cloudwatch.Dimension{
			{Name: aws.String(`InstanceId`), Value: aws.String(`i-0123456789abcdefg`)},
		},
		EvaluationPeriods:       aws.Int64(60),
		InsufficientDataActions: aws.StringSlice([]string{}),
		MetricName:              aws.String(`CPUUtilization`),
		Namespace:               aws.String(`AWS/EC2`),
		OKActions:               aws.StringSlice([]string{}),
		Period:                  aws.Int64(60),
		Statistic:               aws.String(cloudwatch.StatisticAverage),
		Threshold:               aws.Float64(0.0),
		TreatMissingData:        aws.String(`missing`),
	})
	require.NoError(t, err)

	// Verify that the alarm is generated well
	resp, err := svc.DescribeAlarms(&cloudwatch.DescribeAlarmsInput{
		AlarmNames: []*string{&name},
	})
	require.NoError(t, err)
	if len(resp.MetricAlarms) <= 0 {
		t.Fatalf("Error creating Alarm %s", name)
	}
	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

// deleteCloudWatchAlarm is a function to delete the given CloudWatch Alarm.
func deleteCloudWatchAlarm(t *testing.T, svc *cloudwatch.CloudWatch, name *string, checkErr bool) {
	input := &cloudwatch.DeleteAlarmsInput{AlarmNames: []*string{name}}
	_, err := svc.DeleteAlarms(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func assertCloudWatchAlarmsDeleted(t *testing.T, svc *cloudwatch.CloudWatch, identifiers []*string) {
	for _, name := range identifiers {
		resp, err := svc.DescribeAlarms(&cloudwatch.DescribeAlarmsInput{AlarmNames: []*string{name}})
		require.NoError(t, err)
		if len(resp.MetricAlarms) > 0 {
			t.Fatalf("Alarm %s is not deleted", aws.StringValue(name))
		}
	}
}
