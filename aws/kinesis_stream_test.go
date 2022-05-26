package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListKinesisStreams(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := kinesis.New(session)

	sName := createKinesisStream(t, svc)
	defer deleteKinesisStream(t, svc, sName, true)

	sNames, err := getAllKinesisStreams(session, config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(sNames), aws.StringValue(sName))
}

func TestNukeKinesisStreamOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := kinesis.New(session)

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
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := kinesis.New(session)

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

func createKinesisStream(t *testing.T, svc *kinesis.Kinesis) *string {
	uniqueID := util.UniqueID()
	name := fmt.Sprintf("cloud-nuke-test-%s", strings.ToLower(uniqueID))

	_, err := svc.CreateStream(&kinesis.CreateStreamInput{
		ShardCount: aws.Int64(1),
		StreamName: aws.String(name),
	})
	require.NoError(t, err)

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

func deleteKinesisStream(t *testing.T, svc *kinesis.Kinesis, name *string, checkErr bool) {
	_, err := svc.DeleteStream(&kinesis.DeleteStreamInput{
		StreamName: name,
	})
	if checkErr {
		require.NoError(t, err)
	}
}

func assertKinesisStreamsDeleted(t *testing.T, svc *kinesis.Kinesis, identifiers []*string) {
	for _, name := range identifiers {
		stream, err := svc.DescribeStream(&kinesis.DescribeStreamInput{
			StreamName: name,
		})

		// There is an error returned, assert it's because the Stream cannot be found because it's
		// been deleted. Otherwise assert that the stream status is DELETING.
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "ResourceNotFoundException" {
				t.Fatalf("Stream %s is not deleted", aws.StringValue(name))
			}
		} else {
			require.Equal(t, "DELETING", *stream.StreamDescription.StreamStatus)
		}
	}
}
