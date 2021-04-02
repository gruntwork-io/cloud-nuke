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

func createTestQueue(t *testing.T, session *session.Session, name string) *string {
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

	return result.QueueUrl
}

func TestListSqsQueue(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// create 20 test queues, to validate pagination
	queueList := []*string{}
	for n := 0; n < 20; n++ {
		queueName := "cloud-nuke-test-" + util.UniqueID()
		queueUrl := createTestQueue(t, session, queueName)
		require.NoError(t, err)

		queueList = append(queueList, queueUrl)
	}

	// clean up after this test
	defer nukeAllSqsQueues(session, queueList)

	// timestamps to test
	oneHourAgo := time.Now().Add(1 * time.Hour * -1)
	oneHourFromNow := time.Now().Add(1 * time.Hour)

	urls, err := getAllSqsQueue(session, region, oneHourAgo)
	require.NoError(t, err)

	for _, queue := range queueList {
		assert.NotContains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queue))
	}

	urls, err = getAllSqsQueue(session, region, oneHourFromNow)
	require.NoError(t, err)

	for _, queue := range queueList {
		assert.Contains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queue))
	}
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
	queueUrl := createTestQueue(t, session, queueName)
	oneHourFromNow := time.Now().Add(1 * time.Hour)

	urls, err := getAllSqsQueue(session, region, oneHourFromNow)
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queueUrl))

	err = nukeAllSqsQueues(session, []*string{queueUrl})
	require.NoError(t, err)

	urls, err = getAllSqsQueue(session, region, oneHourFromNow)
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(urls), awsgo.StringValue(queueUrl))
}
