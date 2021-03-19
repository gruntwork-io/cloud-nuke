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
	"github.com/gruntwork-io/cloud-nuke/logging"
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

	sleepMessage := "TransitGateway takes some time to create, and since there is no waiter available, we sleep instead."
	sleepFor := 180 * time.Second
	sleepWithMessage(sleepFor, sleepMessage)

	return *result.TransitGateway
}

func TestGetAllTransitGatewayInstances(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	tgwName := "cloud-nuke-test-" + util.UniqueID()
	tgw := createTestTransitGateway(t, session, tgwName)

	ids, err := getAllTransitGatewayInstances(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgw.TransitGatewayId))

	ids, err = getAllTransitGatewayInstances(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgw.TransitGatewayId))

	err = nukeAllTransitGatewayInstances(session, []*string{tgw.TransitGatewayId})
	require.NoError(t, err)
}

func TestNukeTransitGateway(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	require.NoError(t, err)

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
	require.True(t, len(aws.StringValue(result.TransitGatewayRouteTable.TransitGatewayRouteTableId)) > 0, "Could not create test TransitGateway Route Table")

	sleepMessage := "TransitGateway Route Tables takes some time to create, and since there is no waiter available, we sleep instead."
	sleepFor := 180 * time.Second
	sleepWithMessage(sleepFor, sleepMessage)

	return *result.TransitGatewayRouteTable
}

func TestGetAllTransitGatewayRouteTableInstances(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	tgwRouteTableName := "cloud-nuke-test-" + util.UniqueID()
	tgwRouteTable := createTestTransitGatewayRouteTable(t, session, tgwRouteTableName)

	ids, err := getAllTransitGatewayRouteTables(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwRouteTable.TransitGatewayRouteTableId))

	ids, err = getAllTransitGatewayRouteTables(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwRouteTable.TransitGatewayRouteTableId))

	err = nukeAllTransitGatewayRouteTables(session, []*string{tgwRouteTable.TransitGatewayRouteTableId})
	require.NoError(t, err)

	err = nukeAllTransitGatewayInstances(session, []*string{tgwRouteTable.TransitGatewayId})
	require.NoError(t, err)
}

func TestNukeTransitGatewayRouteTable(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	svc := ec2.New(session)

	tgwRouteTableName := "cloud-nuke-test-" + util.UniqueID()
	tgwRouteTable := createTestTransitGatewayRouteTable(t, session, tgwRouteTableName)

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

	err = nukeAllTransitGatewayInstances(session, []*string{tgwRouteTable.TransitGatewayId})
	require.NoError(t, err)
}

func createTestTransitGatewayVpcAttachment(t *testing.T, session *session.Session, name string) ec2.TransitGatewayVpcAttachment {
	svc := ec2.New(session)

	transitGateway := createTestTransitGateway(t, session, name)

	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("isDefault"),
				Values: []*string{awsgo.String("true")},
			},
		},
	}

	vpcs, err := svc.DescribeVpcs(input)
	if err != nil {
		logging.Logger.Error(t, "TransitGatewayVpcAttachment test depends on default VPC availability")
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	assert.NoError(t, err)

	vpc := vpcs.Vpcs[0]

	subnets := getVpcSubnets(t, session, awsgo.StringValue(vpc.VpcId))

	tgwVpctAttachmentName := ec2.TagSpecification{
		ResourceType: awsgo.String(ec2.ResourceTypeTransitGatewayAttachment),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			},
		},
	}

	param := &ec2.CreateTransitGatewayVpcAttachmentInput{
		TagSpecifications: []*ec2.TagSpecification{&tgwVpctAttachmentName},
		TransitGatewayId:  transitGateway.TransitGatewayId,
		VpcId:             vpc.VpcId,
		SubnetIds:         awsgo.StringSlice(subnets),
	}

	result, err := svc.CreateTransitGatewayVpcAttachment(param)
	require.NoError(t, err)
	require.True(t, len(aws.StringValue(result.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)) > 0, "Could not create test Transitgateway Vpc Attachment")

	sleepMessage := "TransitGateway Vpc Attachment takes some time to create, and since there is no waiter available, we sleep instead."
	sleepFor := 180 * time.Second
	sleepWithMessage(sleepFor, sleepMessage)

	return *result.TransitGatewayVpcAttachment
}

func TestGetAllTransitGatewayVpcAttachment(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	tgwName := "cloud-nuke-test-" + util.UniqueID()
	tgwAttachment := createTestTransitGatewayVpcAttachment(t, session, tgwName)

	ids, err := getAllTransitGatewayVpcAttachments(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwAttachment.TransitGatewayAttachmentId))

	ids, err = getAllTransitGatewayVpcAttachments(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwAttachment.TransitGatewayAttachmentId))

	err = nukeAllTransitGatewayVpcAttachments(session, []*string{tgwAttachment.TransitGatewayAttachmentId})
	assert.NoError(t, err)

	err = nukeAllTransitGatewayInstances(session, []*string{tgwAttachment.TransitGatewayId})
	assert.NoError(t, err)
}

func TestNukeTransitGatewayVpcAttachment(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	svc := ec2.New(session)

	tgwVpcAttachmentName := "cloud-nuke-test-" + util.UniqueID()
	tgwVpcAttachment := createTestTransitGatewayVpcAttachment(t, session, tgwVpcAttachmentName)
	_, err = svc.DescribeTransitGatewayVpcAttachments(&ec2.DescribeTransitGatewayVpcAttachmentsInput{
		TransitGatewayAttachmentIds: []*string{
			tgwVpcAttachment.TransitGatewayAttachmentId,
		},
	})
	require.NoError(t, err)

	err = nukeAllTransitGatewayVpcAttachments(session, []*string{tgwVpcAttachment.TransitGatewayAttachmentId})
	require.NoError(t, err)

	ids, err := getAllTransitGatewayVpcAttachments(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), aws.StringValue(tgwVpcAttachment.TransitGatewayAttachmentId))

	err = nukeAllTransitGatewayInstances(session, []*string{tgwVpcAttachment.TransitGatewayId})
	require.NoError(t, err)
}
