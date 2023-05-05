package aws

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Macie tests are limited to testing the ability to find and disable basic Macie
// features. The functionality of cloud-nuke disassociating/deleting members and
// disassociating administrator accounts requires the use of multiple AWS accounts and the
// ability to send and accept invitations within those accounts.

func TestMacie(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)
	logging.Logger.Infof("Region: %s", region)

	region = "us-east-1"

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := macie2.New(session)

	// Check if Macie is enabled
	_, err = svc.GetMacieSession(&macie2.GetMacieSessionInput{})

	if err != nil {
		// GetMacieSession throws an error if Macie is not enabled
		if strings.Contains(err.Error(), "Macie is not enabled") {
			logging.Logger.Infof("Macie not enabled.")
			logging.Logger.Infof("Enabling Macie")
			_, err := svc.EnableMacie(&macie2.EnableMacieInput{})
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	} else {
		logging.Logger.Infof("Macie already enabled")
	}

	macieEnabled, err := getMacie(session, time.Now())
	require.NoError(t, err)

	logging.Logger.Infof("Nuking Macie")
	nukeMacie(session, macieEnabled)

	macieEnabled, err = getMacie(session, time.Now())
	require.NoError(t, err)
	assert.Empty(t, macieEnabled)
}
