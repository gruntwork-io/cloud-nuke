package aws

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Security Hub tests are limited to testing the ability to find and disable security hub
// basic features. The functionality of cloud-nuke disassociating/deleting members and
// disassociating administrator accounts requires the use of multiple AWS accounts and the
// ability to send and accept invitations within those accounts.

func TestSecurityHub(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)
	logging.Logger.Infof("Region: %s", region)

	region = "us-east-1"

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := securityhub.New(session)

	// Check if Security Hub is enabled
	_, err = svc.DescribeHub(&securityhub.DescribeHubInput{})

	if err != nil {
		// DescribeHub throws an error if Security Hub is not enabled
		if strings.Contains(err.Error(), "is not subscribed to AWS Security Hub") {
			logging.Logger.Infof("Security Hub not enabled.")
			logging.Logger.Infof("Enabling Security Hub")
			_, err := svc.EnableSecurityHub(&securityhub.EnableSecurityHubInput{})
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	} else {
		logging.Logger.Infof("Security Hub already enabled")
	}

	hubArns, err := getAllSecurityHubArns(session, time.Now())
	require.NoError(t, err)

	logging.Logger.Infof("Nuking security hub")
	require.NoError(t, nukeSecurityHub(session, hubArns))

	hubArns, err = getAllSecurityHubArns(session, time.Now())
	require.NoError(t, err)
	assert.Empty(t, hubArns)
}
