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
	// Currently we hardcode to region us-east-1, because this is where our "standing" test invite exists
	region := "us-east-1"
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err)

	acceptTestInvite(t, session)
	// Clean up after test by deleting the macie account association
	defer nukeAllMacieAccounts(session, []string{accountId})

	retrievedAccountIds, lookupErr := getAllMacieAccounts(session, time.Now(), config.Config{})
	require.NoError(t, lookupErr)

	assert.Contains(t, retrievedAccountIds, accountId)
}

// TODO - for this to work properly, we'd probably need a standing invite from another test account
// that we can continually re-accept at testing time. In testing with the AWS console, you can contiunously
// delete your membership account association, then re-accept the same standing invitation after enabling Macie again
func acceptTestInvite(t *testing.T, session *session.Session) {
	svc := macie2.New(session)

	// Accept the "standing" invite from our other test account to become a Macie member account
	// This works because Macie invites don't expire or get deleted when you disassociate your member account following an invitation
	acceptInviteInput := &macie2.AcceptInvitationInput{
		AdministratorAccountId: aws.String("087285199408"),                     // phxdevops
		InvitationId:           aws.String("28c0eacd402dd97cbf8a0c14b6cc3237"), // "standing" test invite ID
	}

	_, acceptInviteErr := svc.AcceptInvitation(acceptInviteInput)
	require.NoError(t, acceptInviteErr)
}
