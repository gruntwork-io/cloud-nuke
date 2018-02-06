package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/aws-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestELBv2(t *testing.T, session *session.Session, name string) elbv2.LoadBalancer {
	svc := elbv2.New(session)

	subnetOutput, err := ec2.New(session).DescribeSubnets(&ec2.DescribeSubnetsInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if len(subnetOutput.Subnets) < 2 {
		assert.Fail(t, "Needs at least 2 subnets to create ELBv2")
	}

	subnet1 := *subnetOutput.Subnets[0]
	subnet2 := *subnetOutput.Subnets[1]

	param := &elbv2.CreateLoadBalancerInput{
		Name: awsgo.String(name),
		Subnets: []*string{
			subnet1.SubnetId,
			subnet2.SubnetId,
		},
	}

	result, err := svc.CreateLoadBalancer(param)

	if err != nil {
		assert.Failf(t, "Could not create test ELBv2: %s", errors.WithStackTrace(err).Error())
	}

	if len(result.LoadBalancers) == 0 {
		assert.Failf(t, "Could not create test ELBv2: %s", errors.WithStackTrace(err).Error())
	}

	balancer := *result.LoadBalancers[0]

	err = svc.WaitUntilLoadBalancerAvailable(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{balancer.LoadBalancerArn},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return balancer
}

func TestListELBv2(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbName := "aws-nuke-test-" + util.UniqueID()
	balancer := createTestELBv2(t, session, elbName)

	arns, err := getAllElbv2Instances(session, region)
	if err != nil {
		assert.Fail(t, "Unable to fetch list of v2 ELBs")
	}

	assert.Contains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))

	// clean up after this test
	defer nukeAllElbv2Instances(session, []*string{balancer.LoadBalancerArn})
}

func TestNukeELBv2(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := elbv2.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbName := "aws-nuke-test-" + util.UniqueID()
	balancer := createTestELBv2(t, session, elbName)

	_, err = svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{
			balancer.LoadBalancerArn,
		},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllElbv2Instances(session, []*string{balancer.LoadBalancerArn}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilLoadBalancersDeleted(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{balancer.LoadBalancerArn},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	arns, err := getAllElbv2Instances(session, region)
	if err != nil {
		assert.Fail(t, "Unable to fetch list of v2 ELBs")
	}

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))
}
