package aws

import (
	"fmt"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIamUsers(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// TODO: Implement exclusion by time filter
	// userNames, err := getAllIamUsers(session, region, time.Now().Add(1*time.Hour*-1))
	userNames, err := getAllIamUsers(session, region)
	require.NoError(t, err)

	// TODO: Remove this, just for temporary visual confirmation
	for _, name := range userNames {
		fmt.Printf("this is the name: %s\n", awsgo.StringValue(name))
	}

	assert.NotEmpty(t, userNames)
}
