package aws

import (
	"fmt"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

// createTestEc2KeyPair is a helper method to create a test ec2 key pair
func createTestEc2KeyPair(t *testing.T, svc *ec2.EC2) *ec2.CreateKeyPairOutput {
	keyPair, err := svc.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: awsgo.String(util.UniqueID()),
	})

	require.NoError(t, err)

	err = svc.WaitUntilKeyPairExists(&ec2.DescribeKeyPairsInput{
		KeyPairIds: awsgo.StringSlice([]string{*keyPair.KeyPairId}),
	})

	require.NoError(t, err)
	return keyPair
}

func TestEc2KeyPairListAndNuke(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	testSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	svc := ec2.New(testSession)
	createdKeyPair := createTestEc2KeyPair(t, svc)
	testExcludeAfterTime := time.Now().Add(24 * time.Hour)
	keyPairIds, err := getAllEc2KeyPairs(testSession, testExcludeAfterTime, config.Config{})

	assert.Contains(t, awsgo.StringValueSlice(keyPairIds), *createdKeyPair.KeyPairId)

	// Note: nuking the ec2 key pair created for testing purpose
	err = nukeAllEc2KeyPairs(testSession, []*string{createdKeyPair.KeyPairId})
	require.NoError(t, err)

	// Check whether the key still exist or not.
	keyPairIds, err = getAllEc2KeyPairs(testSession, testExcludeAfterTime, config.Config{})
	require.NoError(t, err)
	require.NotContains(t, awsgo.StringValueSlice(keyPairIds), *createdKeyPair.KeyPairId)
}

func TestEc2KeyPairListWithConfig(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	region, err := getRandomRegion()
	require.NoError(t, err)

	testSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	svc := ec2.New(testSession)
	createdKeyPair := createTestEc2KeyPair(t, svc)
	createdKeyPair2 := createTestEc2KeyPair(t, svc)

	// Regex expression to not include first key pair
	nameRegexExp, err := regexp.Compile(fmt.Sprintf("^%s*", *createdKeyPair.KeyName))
	excludeConfig := config.Config{
		EC2KeyPair: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *nameRegexExp,
					},
				},
			},
		},
	}

	testExcludeAfterTime := time.Now().Add(24 * time.Hour)
	keyPairIds, err := getAllEc2KeyPairs(testSession, testExcludeAfterTime, excludeConfig)
	assert.NotContains(t, awsgo.StringValueSlice(keyPairIds), *createdKeyPair.KeyPairId)
	assert.Contains(t, awsgo.StringValueSlice(keyPairIds), *createdKeyPair2.KeyPairId)
}
