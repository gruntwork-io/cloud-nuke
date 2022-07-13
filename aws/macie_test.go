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
	defer nukeAllMacieMemberAccounts(session, []string{accountId})

	retrievedAccountIds, lookupErr := getAllMacieMemberAccounts(session, time.Now(), config.Config{})
	require.NoError(t, lookupErr)

	assert.Contains(t, retrievedAccountIds, accountId)
}

// Macie is not very conducive to programmatic testing. In order to make this test work, we  maintain a standing invite
// from our phxdevops test account to our nuclear-wasteland account. We can continuously "nuke" our membership because
// Macie supports a member account *that was invited* to remove its own association at any time. Meanwhile, diassociating
// in this manner does not destroy or invalidate the original invitation, which allows us to to continually re-accept it
// from our nuclear-wasteland account (where cloud-nuke tests are run), just so that we can nuke it again
//
// Macie is also regional, so for the purposes of cost-savings and lower admin overhead, we're initially only testing this
// in the one hardcoded region
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
