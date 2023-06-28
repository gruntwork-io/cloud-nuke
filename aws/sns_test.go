package aws

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"

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

func createTestSNSTopic(t *testing.T, session *session.Session, name string, setFirstSeenTag bool) (*TestSNSTopic, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)

	svc := sns.NewFromConfig(cfg)

	param := &sns.CreateTopicInput{
		Name: &name,
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

	if setFirstSeenTag {
		// Set the first seen tag on the SNS Topic
		err := setFirstSeenSNSTopicTag(context.TODO(), svc, *output.TopicArn, firstSeenTagKey, time.Now())
		if err != nil {
			return nil, err
		}
	}

	return &TestSNSTopic{
		Name: param.Name,
		Arn:  output.TopicArn,
	}, nil
}

func TestListSNSTopics(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	testSNSTopic, createTestSNSTopicErr := createTestSNSTopic(t, session, snsTopicName, true)
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
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	snsTopicName := "aws-nuke-test-" + util.UniqueID()

	testSNSTopic, createTestSNSTopicErr := createTestSNSTopic(t, session, snsTopicName, true)
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

	testSNSTopic, createTestErr := createTestSNSTopic(t, session, testSNSTopicName, true)
	require.NoError(t, createTestErr)
	testSNSTopic2, createTestErr2 := createTestSNSTopic(t, session, testSNSTopicName2, true)
	require.NoError(t, createTestErr2)

	nukeErr := nukeAllSNSTopics(session, []*string{testSNSTopic.Arn, testSNSTopic2.Arn})
	require.NoError(t, nukeErr)

	// Make sure the SNS topics were deleted
	snsTopicArns, err := getAllSNSTopics(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(snsTopicArns), aws.StringValue(testSNSTopic.Arn))
	assert.NotContains(t, aws.StringValueSlice(snsTopicArns), aws.StringValue(testSNSTopic2.Arn))
}

func TestNukeSNSTopicWithFilter(t *testing.T) {
	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	testSNSTopicName := "aws-nuke-test-" + util.UniqueID()
	testSNSTopicName2 := "aws-do-not-nuke-test-" + util.UniqueID()

	testSNSTopic, createTestErr := createTestSNSTopic(t, session, testSNSTopicName, true)
	require.NoError(t, createTestErr)

	testSNSTopic2, createTestErr2 := createTestSNSTopic(t, session, testSNSTopicName2, true)
	require.NoError(t, createTestErr2)

	// as sns topics are online, lets clean up after this test
	defer nukeAllSNSTopics(session, []*string{testSNSTopic.Arn, testSNSTopic2.Arn})

	topics, err := getAllSNSTopics(session, time.Now(), config.Config{
		SNS: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("aws-do-not-nuke-test-.*")}},
			},
		},
	})
	require.NoError(t, err)

	// with the filters, we expect to only see the testSNSTopic in the findings, and not the testSNSTopic2
	assert.NotContains(t, aws.StringValueSlice(topics), aws.StringValue(testSNSTopic2.Arn))
	assert.Contains(t, aws.StringValueSlice(topics), aws.StringValue(testSNSTopic.Arn))
}

func TestSNSFirstSeenTagLogicIsCorrect(t *testing.T) {
	ctx := context.Background()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	snsTopicName := "aws-nuke-test-" + util.UniqueID()
	testSNSTopic, createTestSNSTopicErr := createTestSNSTopic(t, session, snsTopicName, false)
	require.NoError(t, createTestSNSTopicErr)

	// clean up after this test
	defer nukeAllSNSTopics(session, []*string{testSNSTopic.Arn})

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)

	svc := sns.NewFromConfig(cfg)

	// check that the first seen tag is not present
	firstSeen, err := getFirstSeenSNSTopicTag(ctx, svc, *testSNSTopic.Arn, firstSeenTagKey)
	require.NoError(t, err)
	assert.Nil(t, firstSeen)

	// update the first seen tag
	now := time.Now().UTC()
	err = setFirstSeenSNSTopicTag(ctx, svc, *testSNSTopic.Arn, firstSeenTagKey, now)
	require.NoError(t, err)

	// check that the first seen tag was updated
	firstSeen, err = getFirstSeenSNSTopicTag(ctx, svc, *testSNSTopic.Arn, firstSeenTagKey)
	require.NoError(t, err)

	// We lose some precision when we tag the resource with the time due to the format, so to compare like for like,
	// cast both to the same string format, which is also the same format used by the firstSeenSNSTopicTag function
	assert.Equal(t, now.Format(firstSeenTimeFormat), firstSeen.Format(firstSeenTimeFormat))
}

func TestNukeSNSTopicWithTimeExclusion(t *testing.T) {
	ctx := context.Background()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	snsTopicName := "aws-nuke-test-" + util.UniqueID()
	testSNSTopic, createTestSNSTopicErr := createTestSNSTopic(t, session, snsTopicName, true)
	require.NoError(t, createTestSNSTopicErr)

	// clean up after this test
	defer nukeAllSNSTopics(session, []*string{testSNSTopic.Arn})

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)
	svc := sns.NewFromConfig(cfg)

	// update the tag on the sns topic to be 1 in the future
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	err = setFirstSeenSNSTopicTag(ctx, svc, *testSNSTopic.Arn, firstSeenTagKey, oneHourAgo)
	require.NoError(t, err)

	// ensure the sns topic is not found when we search for sns topics created, with a time exclusion
	// that is 2 hours ago
	twoHoursAgo := oneHourAgo.Add(-1 * time.Hour)
	topics, err := getAllSNSTopics(session, twoHoursAgo, config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(topics), aws.StringValue(testSNSTopic.Arn))
}

func TestShouldIncludeSNS(t *testing.T) {
	topic_name := func(name string) string {
		return "arn:aws:sns:us-east-1:123456789012:" + name
	}

	tests := map[string]struct {
		excludeAfter  time.Time
		firstSeenTime time.Time
		config        config.Config
		TopicArn      string
		Expected      bool
	}{
		"should include sns topic when no config is provided": {
			TopicArn: topic_name("test-topic"),
			Expected: true,
		},
		"should not include sns topic when name matches excludes": {
			TopicArn: topic_name("test-topic"),
			config: config.Config{
				SNS: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-topic")}},
					},
				},
			},
			Expected: false,
		},
		"should include sns topic when name matches includes": {
			TopicArn: topic_name("test-topic"),
			config: config.Config{
				SNS: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-topic")}},
					},
				},
			},
			Expected: true,
		},
		"should not include sns topic when name matches excludes and includes": {
			TopicArn: topic_name("test-topic"),
			config: config.Config{
				SNS: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-topic")}},
					},
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-topic")}},
					},
				},
			},
			Expected: false,
		},
		"should not include sns topic when excludes after is before first seen time": {
			TopicArn:      topic_name("test-topic"),
			excludeAfter:  time.Now().UTC().Add(-1 * time.Hour),
			firstSeenTime: time.Now().UTC(),
			Expected:      false,
		},
		"should include sns topic when excludes after is after first seen time": {
			TopicArn:      topic_name("test-topic"),
			excludeAfter:  time.Now().UTC().Add(1 * time.Hour),
			firstSeenTime: time.Now().UTC(),
			Expected:      true,
		},
		"should not include sns topic when excludes after is before first seen, but config includes the name": {
			TopicArn:      topic_name("test-topic"),
			excludeAfter:  time.Now().UTC().Add(-1 * time.Hour),
			firstSeenTime: time.Now().UTC(),
			config: config.Config{
				SNS: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-topic")}},
					},
				},
			},
			Expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shouldInclude := shouldIncludeSNS(test.TopicArn, test.excludeAfter, test.firstSeenTime, test.config)
			assert.Equal(t, test.Expected, shouldInclude)
		})
	}
}
