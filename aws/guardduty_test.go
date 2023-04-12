package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestGuardDutyDetector(t *testing.T, session *session.Session) string {
	svc := guardduty.New(session)

	param := &guardduty.CreateDetectorInput{
		ClientToken: aws.String("test-value-idempotent-token"),
		DataSources: &guardduty.DataSourceConfigurations{
			Kubernetes: &guardduty.KubernetesConfiguration{
				AuditLogs: &guardduty.KubernetesAuditLogsConfiguration{
					Enable: aws.Bool(false),
				},
			},
			S3Logs: &guardduty.S3LogsConfiguration{
				Enable: aws.Bool(false),
			},
		},
		Enable:                     awsgo.Bool(true),
		FindingPublishingFrequency: nil,
		Tags: map[string]*string{
			"test-detector": aws.String("true"),
		},
	}

	detectorOutput, err := svc.CreateDetector(param)
	if err != nil {
		assert.Failf(t, "Could not create test GuardDuty Detector", errors.WithStackTrace(err).Error())
	}

	return awsgo.StringValue(detectorOutput.DetectorId)
}

func TestListGuardDutyDetectors(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
	)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	testDetectorId := createTestGuardDutyDetector(t, session)

	// clean up after this test
	defer nukeAllGuardDutyDetectors(session, []string{testDetectorId})

	detectorIds, lookupErr := getAllGuardDutyDetectors(session, time.Now(), config.Config{}, GuardDuty{}.MaxBatchSize())

	require.NoError(t, lookupErr)

	if err != nil {
		assert.Fail(t, "Unable to fetch list of GuardDuty Detector Ids")
	}

	assert.Contains(t, detectorIds, testDetectorId)
}

func TestTimeFilterExclusionNewlyCreatedGuardDutyDetector(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	testDetectorId := createTestGuardDutyDetector(t, session)
	// Clean up after this test
	defer nukeAllGuardDutyDetectors(session, []string{testDetectorId})

	// Assert detectors are picked up without filters
	detectorIds, err := getAllGuardDutyDetectors(session, time.Now(), config.Config{}, GuardDuty{}.MaxBatchSize())
	require.NoError(t, err)
	assert.Contains(t, detectorIds, testDetectorId)

	// Assert detector doesn't appear when we look at detectors older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	detectorIdsOlder, err := getAllGuardDutyDetectors(session, olderThan, config.Config{}, GuardDuty{}.MaxBatchSize())
	require.NoError(t, err)
	assert.NotContains(t, detectorIdsOlder, testDetectorId)
}

func TestNukeGuardDutyDetectorOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	testDetectorId := createTestGuardDutyDetector(t, session)

	// Make sure the GuardDuty test detector was created and can be listed
	detectorIdsPreNuke, lookupErr := getAllGuardDutyDetectors(session, time.Now(), config.Config{}, GuardDuty{}.MaxBatchSize())
	require.NoError(t, lookupErr)

	require.Equal(t, 1, len(detectorIdsPreNuke))

	identifiers := []string{testDetectorId}

	require.NoError(
		t,
		nukeAllGuardDutyDetectors(session, identifiers),
	)

	// Make sure the GuardDuty detector was deleted
	detectorIdsPostNuke, secondLookupErr := getAllGuardDutyDetectors(session, time.Now(), config.Config{}, GuardDuty{}.MaxBatchSize())
	require.NoError(t, secondLookupErr)
	require.Equal(t, 0, len(detectorIdsPostNuke))
}

// TestNukeGuardDutyDetectorMoreThanOne verifies that you can create and nuke multiple detectors in different regions simultaneously
func TestNukeGuardDutyDetectorMoreThanOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region1, err := getRandomRegion()
	require.NoError(t, err)

	session1, createSessionErr := session.NewSession(&aws.Config{Region: aws.String(region1)})
	require.NoError(t, createSessionErr)

	testDetectorId1 := createTestGuardDutyDetector(t, session1)

	region2, err := getRandomRegionWithExclusions([]string{region1})
	require.NoError(t, err)

	session2, createSessionErr2 := session.NewSession(&aws.Config{Region: aws.String(region2)})
	require.NoError(t, createSessionErr2)

	testDetectorId2 := createTestGuardDutyDetector(t, session2)

	require.NoError(
		t,
		nukeAllGuardDutyDetectors(session1, []string{testDetectorId1}),
	)

	require.NoError(
		t,
		nukeAllGuardDutyDetectors(session2, []string{testDetectorId2}),
	)

	// Make sure the GuardDuty detector was deleted
	detectorIdsPostNuke1, lookupErr := getAllGuardDutyDetectors(session1, time.Now(), config.Config{}, GuardDuty{}.MaxBatchSize())
	require.NoError(t, lookupErr)
	require.Equal(t, 0, len(detectorIdsPostNuke1))

	// Make sure the GuardDuty detector was deleted
	detectorIdsPostNuke2, lookupErr := getAllGuardDutyDetectors(session2, time.Now(), config.Config{}, GuardDuty{}.MaxBatchSize())
	require.NoError(t, lookupErr)
	require.Equal(t, 0, len(detectorIdsPostNuke2))
}
