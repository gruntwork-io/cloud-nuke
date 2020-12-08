package aws

import (
	"fmt"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func TestListIamUsers(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// TODO: Implement exclusion by time filter
	// userNames, err := getAllIamUsers(session, region, time.Now().Add(1*time.Hour*-1))
	userNames, err := getAllIamUsers(session, region)
	if err != nil {
		assert.Fail(t, "Unable to fetch list of IAM users")
	}

	// TODO: Remove this, just for temporary visual confirmation
	for _, name := range userNames {
		fmt.Printf("this is the *name: %s\n", *name)
	}

	assert.NotEmpty(t, userNames)
}
