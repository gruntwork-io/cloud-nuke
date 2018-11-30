package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getSubnetsInDifferentAZs(t *testing.T, session *session.Session) (*ec2.Subnet, *ec2.Subnet) {
	subnetOutput, err := ec2.New(session).DescribeSubnets(&ec2.DescribeSubnetsInput{})
	require.NoError(t, err)
	require.True(t, len(subnetOutput.Subnets) >= 2)

	subnet1 := subnetOutput.Subnets[0]

	for i := 1; i < len(subnetOutput.Subnets); i++ {
		subnet2 := subnetOutput.Subnets[i]
		if *subnet1.AvailabilityZone != *subnet2.AvailabilityZone && *subnet1.SubnetId != *subnet2.SubnetId && *subnet1.VpcId == *subnet2.VpcId {
			return subnet1, subnet2
		}
	}

	require.Fail(t, "Unable to find 2 subnets in different Availability Zones")
	return nil, nil
}

func createTestELBv2(t *testing.T, session *session.Session, name string) elbv2.LoadBalancer {
	svc := elbv2.New(session)

	subnet1, subnet2 := getSubnetsInDifferentAZs(t, session)

	param := &elbv2.CreateLoadBalancerInput{
		Name: awsgo.String(name),
		Subnets: []*string{
			subnet1.SubnetId,
			subnet2.SubnetId,
		},
	}

	result, err := svc.CreateLoadBalancer(param)
	require.NoError(t, err)
	require.Equal(t, len(result.LoadBalancers), 0)

	balancer := *result.LoadBalancers[0]

	err = svc.WaitUntilLoadBalancerAvailable(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{balancer.LoadBalancerArn},
	})
	require.NoError(t, err)

	return balancer
}

func TestListELBv2(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	elbName := "cloud-nuke-test-" + util.UniqueID()
	balancer := createTestELBv2(t, session, elbName)
	// clean up after this test
	defer nukeAllElbv2Instances(session, []*string{balancer.LoadBalancerArn})

	arns, err := getAllElbv2Instances(session, region, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))

	arns, err = getAllElbv2Instances(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))
}

func TestNukeELBv2(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	svc := elbv2.New(session)

	elbName := "cloud-nuke-test-" + util.UniqueID()
	balancer := createTestELBv2(t, session, elbName)

	_, err = svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{
			balancer.LoadBalancerArn,
		},
	})
	require.NoError(t, err)

	err = nukeAllElbv2Instances(session, []*string{balancer.LoadBalancerArn})
	require.NoError(t, err)

	err = svc.WaitUntilLoadBalancersDeleted(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{balancer.LoadBalancerArn},
	})
	require.NoError(t, err)

	arns, err := getAllElbv2Instances(session, region, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))
}
