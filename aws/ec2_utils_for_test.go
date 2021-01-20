package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/require"
)

func getVpcSubnets(t *testing.T, session *session.Session, vpcId string) []string {
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

	var subnets []string

	for _, v := range result.Subnets {
		subnets = append(subnets, awsgo.StringValue(v.SubnetId))
	}
	return subnets
}
