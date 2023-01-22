package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/require"
)

func getVpcSubnetsDistinctByAz(t *testing.T, session *session.Session, vpcId string) []string {
	result := getVpcSubnets(t, session, vpcId)
	var subnetsByAz = make(map[string]string)
	// collect subnetworks distinct by AZ
	for _, v := range result.Subnets {
		subnetsByAz[*v.AvailabilityZone] = awsgo.StringValue(v.SubnetId)
	}
	var subnets []string
	for _, subnet := range subnetsByAz {
		subnets = append(subnets, subnet)
	}
	return subnets
}

func getVpcSubnets(t *testing.T, session *session.Session, vpcId string) *ec2.DescribeSubnetsOutput {
	svc := ec2.New(session)

	param := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{vpcId}),
			},
		},
	}

	result, err := svc.DescribeSubnets(param)
	require.NoError(t, err)
	return result
}
