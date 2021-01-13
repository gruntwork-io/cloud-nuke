package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTransitGatewayRouteTable(t *testing.T, session *session.Session, name string) ec2.TransitGatewayRouteTable {
	svc := ec2.New(session)

	transitGateway := createTestTransitGateway(t, session, name)

	tgwRouteTableName := ec2.TagSpecification{
		ResourceType: awsgo.String(ec2.ResourceTypeTransitGatewayRouteTable),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			},
		},
	}

	param := &ec2.CreateTransitGatewayRouteTableInput{
		TagSpecifications: []*ec2.TagSpecification{&tgwRouteTableName},
		TransitGatewayId:  transitGateway.TransitGatewayId,
	}

	result, err := svc.CreateTransitGatewayRouteTable(param)
	require.NoError(t, err)
	require.True(t, len(aws.StringValue(result.TransitGatewayRouteTable.TransitGatewayRouteTableId)) > 0, "Could not create test TransitGatewayRouteTable")

	//TransitGateway takes some time to be available and there isn't Waiters available yet
	//To avoid test errors, I'm introducing a sleep call
	time.Sleep(180 * time.Second)
	tgwRouteTable := *result.TransitGatewayRouteTable
	return tgwRouteTable
}

func TestGetAllTransitGatewayRouteTableInstances(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	tgwRouteTableName := "clud-nuke-test-" + util.UniqueID()
	tgwRouteTable := createTestTransitGatewayRouteTable(t, session, tgwRouteTableName)

	defer nukeAllTransitGatewayRouteTables(session, []*string{tgwRouteTable.TransitGatewayRouteTableId})
	defer nukeAllTransitGatewayInstances(session, []*string{tgwRouteTable.TransitGatewayId})

	ids, err := getAllTransitGatewayRouteTables(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwRouteTable.TransitGatewayRouteTableId))

	ids, err = getAllTransitGatewayRouteTables(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwRouteTable.TransitGatewayRouteTableId))
}

func TestNukeTransitGatewayRouteTable(t *testing.T) {
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

	tgwRouteTableName := "clud-nuke-test-" + util.UniqueID()
	tgwRouteTable := createTestTransitGatewayRouteTable(t, session, tgwRouteTableName)
	defer nukeAllTransitGatewayInstances(session, []*string{tgwRouteTable.TransitGatewayId})

	_, err = svc.DescribeTransitGatewayRouteTables(&ec2.DescribeTransitGatewayRouteTablesInput{
		TransitGatewayRouteTableIds: []*string{
			tgwRouteTable.TransitGatewayRouteTableId,
		},
	})
	require.NoError(t, err)

	err = nukeAllTransitGatewayRouteTables(session, []*string{tgwRouteTable.TransitGatewayRouteTableId})
	require.NoError(t, err)

	ids, err := getAllTransitGatewayRouteTables(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwRouteTable.TransitGatewayRouteTableId))
}
