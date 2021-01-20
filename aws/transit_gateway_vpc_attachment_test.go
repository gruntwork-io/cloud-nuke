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

	vpcs, _ := svc.DescribeVpcs(input)
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
	require.True(t, len(aws.StringValue(result.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)) > 0, "Could not create test Transitgateway Attachment")

	//TransitGateway takes some time to be available and there isn't Waiters available yet
	//To avoid test errors, I'm introducing a sleep call
	time.Sleep(180 * time.Second)
	tgwAttachment := *result.TransitGatewayVpcAttachment
	return tgwAttachment
}

func TestGetAllTransitGatewayVpcAttachment(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	tgwName := "clud-nuke-test-" + util.UniqueID()
	tgwAttachment := createTestTransitGatewayVpcAttachment(t, session, tgwName)

	ids, err := getAllTransitGatewayVpcAttachments(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwAttachment.TransitGatewayAttachmentId))

	ids, err = getAllTransitGatewayVpcAttachments(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(ids), awsgo.StringValue(tgwAttachment.TransitGatewayAttachmentId))

	nukeAllTransitGatewayVpcAttachments(session, []*string{tgwAttachment.TransitGatewayAttachmentId})
	nukeAllTransitGatewayInstances(session, []*string{tgwAttachment.TransitGatewayId})
}

func TestNukeTransitGatewayVpcAttachment(t *testing.T) {
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

	tgwVpcAttachmentName := "cloud-nuke-test-" + util.UniqueID()
	tgwVpcAttachment := createTestTransitGatewayVpcAttachment(t, session, tgwVpcAttachmentName)
	_, err = svc.DescribeTransitGatewayVpcAttachments(&ec2.DescribeTransitGatewayVpcAttachmentsInput{
		TransitGatewayAttachmentIds: []*string{
			tgwVpcAttachment.TransitGatewayAttachmentId,
		},
	})
	require.NoError(t, err)
	defer nukeAllTransitGatewayInstances(session, []*string{tgwVpcAttachment.TransitGatewayId})

	err = nukeAllTransitGatewayVpcAttachments(session, []*string{tgwVpcAttachment.TransitGatewayAttachmentId})
	require.NoError(t, err)

	ids, err := getAllTransitGatewayVpcAttachments(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(ids), aws.StringValue(tgwVpcAttachment.TransitGatewayAttachmentId))
}
