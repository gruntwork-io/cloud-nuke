package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMacieAccounts(t *testing.T) {
	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err)

	enableMacieAndAcceptInvite(t, session)
	// Clean up after test by deleting the macie account association
	defer nukeAllMacieAccounts(session, []string{accountId})

	retrievedAccountIds, lookupErr := getAllMacieAccounts(session, time.Now(), config.Config{})
	require.NoError(t, lookupErr)

	assert.Contains(t, retrievedAccountIds, accountId)
}

// TODO - for this to work properly, we'd probably need a standing invite from another test account
// that we can continually re-accept at testing time. In testing with the AWS console, you can contiunously
// delete your membership account association, then re-accept the same standing invitation after enabling Macie again
func enableMacieAndAcceptInvite(t *testing.T, session *session.Session) {
	svc := macie2.New(session)

	_, err := svc.EnableMacie(&macie2.EnableMacieInput{})
	require.NoError(t, err)
}
