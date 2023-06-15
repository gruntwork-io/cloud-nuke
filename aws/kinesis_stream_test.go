package aws

import (
	"context"
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"strings"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListKinesisStreams(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)

	svc := kinesis.NewFromConfig(cfg)

	sName := createKinesisStream(t, svc)
	defer deleteKinesisStream(t, svc, sName, true)

	sNames, err := getAllKinesisStreams(session, config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(sNames), aws.StringValue(sName))
}

func TestNukeKinesisStreamOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)

	svc := kinesis.NewFromConfig(cfg)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	sName := createKinesisStream(t, svc)
	defer deleteKinesisStream(t, svc, sName, true)
	identifiers := []*string{sName}

	require.NoError(
		t,
		nukeAllKinesisStreams(session, identifiers),
	)

	assertKinesisStreamsDeleted(t, svc, identifiers)
}

func TestNukeKinesisStreamMoreThanOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	require.NoError(t, err)

	svc := kinesis.NewFromConfig(cfg)

	sNames := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		sName := createKinesisStream(t, svc)
		defer deleteKinesisStream(t, svc, sName, true)
		sNames = append(sNames, sName)
	}

	require.NoError(
		t,
		nukeAllKinesisStreams(session, sNames),
	)

	assertKinesisStreamsDeleted(t, svc, sNames)
}

func createKinesisStream(t *testing.T, svc *kinesis.Client) *string {
	uniqueID := util.UniqueID()
	name := fmt.Sprintf("cloud-nuke-test-%s", strings.ToLower(uniqueID))

	_, err := svc.CreateStream(context.TODO(), &kinesis.CreateStreamInput{
		ShardCount: aws.Int32(1),
		StreamName: aws.String(name),
	})
	require.NoError(t, err)

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

func deleteKinesisStream(t *testing.T, svc *kinesis.Client, name *string, checkErr bool) {
	_, err := svc.DeleteStream(context.TODO(), &kinesis.DeleteStreamInput{
		StreamName: name,
	})
	if checkErr {
		require.NoError(t, err)
	}
}

func assertKinesisStreamsDeleted(t *testing.T, svc *kinesis.Client, identifiers []*string) {
	for _, name := range identifiers {
		stream, err := svc.DescribeStream(context.TODO(), &kinesis.DescribeStreamInput{
			StreamName: name,
		})

		// There is an error returned, assert it's because the Stream cannot be found because it's
		// been deleted. Otherwise assert that the stream status is DELETING.
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "ResourceNotFoundException" {
				t.Fatalf("Stream %s is not deleted", aws.StringValue(name))
			}
		} else {
			require.Equal(t, types.StreamStatusDeleting, stream.StreamDescription.StreamStatus)
		}
	}
}
