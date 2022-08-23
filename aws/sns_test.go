package aws

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestSNSTopic struct {
	Name *string
	Arn  *string
}

func createTestSNSTopic(t *testing.T, session *session.Session, name string) (*TestSNSTopic, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)

	svc := sns.NewFromConfig(cfg)

	testSNSTopic := &TestSNSTopic{
		Name: aws.String(name),
	}

	param := &sns.CreateTopicInput{
		Name: testSNSTopic.Name,
	}

	// Do a coin-flip to choose either a FIFO or Standard SNS Topic
	coin := []string{
		"true",
		"false",
	}
	rand.Seed(time.Now().UnixNano())
	coinflip := coin[rand.Intn(len(coin))]
	param.Attributes = make(map[string]string)
	param.Attributes["FifoTopic"] = coinflip

	// If we did choose to create a fifo queue, the name must end in ".fifo"
	if coinflip == "true" {
		param.Name = aws.String(fmt.Sprintf("%s.fifo", aws.StringValue(param.Name)))
	}

	output, err := svc.CreateTopic(context.TODO(), param)
	if err != nil {
		assert.Failf(t, "Could not create test SNS Topic: %s", errors.WithStackTrace(err).Error())
	}

	testSNSTopic.Arn = output.TopicArn

	return testSNSTopic, nil
}

func TestListSNSTopics(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
	)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	snsTopicName := "aws-nuke-test-" + util.UniqueID()
	testSNSTopic, createTestSNSTopicErr := createTestSNSTopic(t, session, snsTopicName)
	require.NoError(t, createTestSNSTopicErr)
	// clean up after this test
	defer nukeAllSNSTopics(session, []*string{testSNSTopic.Arn})

	snsTopicArns, err := getAllSNSTopics(session, time.Now(), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of SNS Topics")
	}

	assert.Contains(t, awsgo.StringValueSlice(snsTopicArns), aws.StringValue(testSNSTopic.Arn))
}

func TestNukeSNSTopicOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	snsTopicName := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testSNSTopic, createTestSNSTopicErr := createTestSNSTopic(t, session, snsTopicName)
	require.NoError(t, createTestSNSTopicErr)

	nukeErr := nukeAllSNSTopics(session, []*string{testSNSTopic.Arn})
	require.NoError(t, nukeErr)

	// Make sure the SNS Topic was deleted
	snsTopicArns, err := getAllSNSTopics(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(snsTopicArns), aws.StringValue(testSNSTopic.Arn))
}

func TestNukeSNSTopicMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	testSNSTopicName := "aws-nuke-test-" + util.UniqueID()
	testSNSTopicName2 := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testSNSTopic, createTestErr := createTestSNSTopic(t, session, testSNSTopicName)
	require.NoError(t, createTestErr)
	testSNSTopic2, createTestErr2 := createTestSNSTopic(t, session, testSNSTopicName2)
	require.NoError(t, createTestErr2)

	nukeErr := nukeAllSNSTopics(session, []*string{testSNSTopic.Arn, testSNSTopic2.Arn})
	require.NoError(t, nukeErr)

	// Make sure the SNS topics were deleted
	snsTopicArns, err := getAllSNSTopics(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(snsTopicArns), aws.StringValue(testSNSTopic.Arn))
	assert.NotContains(t, aws.StringValueSlice(snsTopicArns), aws.StringValue(testSNSTopic2.Arn))
}
