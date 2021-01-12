package aws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func createTestTransitGateway(t *testing.T, session *session.Session, name string) ec2.TransitGateway {
	svc := ec2.New(session)

	tgwName := ec2.TagSpecification{
		ResourceType: awsgo.String(ec2.ResourceTypeTransitGateway),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			},
		},
	}

	param := &ec2.CreateTransitGatewayInput{
		TagSpecifications: []*ec2.TagSpecification{&tgwName},
	}

	result, err := svc.CreateTransitGateway(param)
	require.NoError(t, err)
	require.True(t, len(aws.StringValue(result.TransitGateway.TransitGatewayId)) > 0, "Could not create test TransitGateway")

	//TransitGateway takes some time to be available and there isn't Waiters available yet
	//To avoid test errors, I'm introducing a sleep call
	time.Sleep(180 * time.Second)
	tgw := *result.TransitGateway
	return tgw
}

func TestGetAllTransitGatewayInstances(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	tgwName := "clud-nuke-test-" + util.UniqueID()
	tgw := createTestTransitGateway(t, session, tgwName)

	defer nukeAllTransitGatewayInstances(session, []*string{tgw.TransitGatewayId})

	ids, err := getAllTransitGatewayInstances(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgw.TransitGatewayId))

	ids, err = getAllTransitGatewayInstances(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgw.TransitGatewayId))
}

func TestNukeTransitGateway(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	svc := ec2.New(session)

	tgwName := "cloud-nuke-test-" + util.UniqueID()
	tgw := createTestTransitGateway(t, session, tgwName)

	_, err = svc.DescribeTransitGateways(&ec2.DescribeTransitGatewaysInput{
		TransitGatewayIds: []*string{
			tgw.TransitGatewayId,
		},
	})
	require.NoError(t, err)

	err = nukeAllTransitGatewayInstances(session, []*string{tgw.TransitGatewayId})
	require.NoError(t, err)

	ids, err := getAllTransitGatewayInstances(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgw.TransitGatewayId))
}
