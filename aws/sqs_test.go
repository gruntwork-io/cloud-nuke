package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestQueue(t *testing.T, session *session.Session, name string) (*string, error) {
	svc := sqs.New(session)

	param := &sqs.CreateQueueInput{
		QueueName: awsgo.String(name),
		Attributes: map[string]*string{
			"DelaySeconds":           awsgo.String("60"),
			"MessageRetentionPeriod": awsgo.String("86400"),
		},
	}

	result, err := svc.CreateQueue(param)
	require.NoError(t, err)
	require.True(t, len(awsgo.StringValue(result.QueueUrl)) > 0, "Can't create test Sqs Queue")

	return result.QueueUrl, nil
}

func TestListSqsQueue(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	queueName := "cloud-nuke-test-" + util.UniqueID()
	queueUrl, err := createTestQueue(t, session, queueName)
	require.NoError(t, err)

	// clean up after this test
	defer nukeAllSqsQueues(session, []*string{queueUrl})

	urls, err := getAllSqsQueue(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queueUrl))

	urls, err = getAllSqsQueue(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queueUrl))
}

func TestNukeSqsQueue(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	queueName := "cloud-nuke-test-" + util.UniqueID()
	queueUrl, _ := createTestQueue(t, session, queueName)

	err = nukeAllSqsQueues(session, []*string{queueUrl})
	require.NoError(t, err)

	urls, err := getAllSqsQueue(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queueUrl))
}
