package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getSubnetsInDifferentAZs(t *testing.T, session *session.Session) (*ec2.Subnet, *ec2.Subnet) {
	subnetOutput, err := ec2.New(session).DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("default-for-az"),
				Values: []*string{awsgo.String("true")},
			},
		},
	})
	require.NoError(t, err)
	require.True(t, len(subnetOutput.Subnets) >= 2)

	subnet1Idx := -1
	for idx, subnet := range subnetOutput.Subnets {
		if !collections.ListContainsElement(AvailabilityZoneBlackList, awsgo.StringValue(subnet.AvailabilityZone)) {
			subnet1Idx = idx
			break
		}
	}
	if subnet1Idx == -1 {
		require.Fail(t, "Unable to find a subnet in an availability zone that is not blacklisted.")
	}
	subnet1 := subnetOutput.Subnets[subnet1Idx]
	az1 := awsgo.StringValue(subnet1.AvailabilityZone)
	subnet1Id := awsgo.StringValue(subnet1.SubnetId)
	subnet1VpcId := awsgo.StringValue(subnet1.VpcId)

	for i := subnet1Idx + 1; i < len(subnetOutput.Subnets); i++ {
		subnet2 := subnetOutput.Subnets[i]
		az2 := awsgo.StringValue(subnet2.AvailabilityZone)
		if collections.ListContainsElement(AvailabilityZoneBlackList, az2) {
			// Skip because subnet is in a blacklisted AZ
			continue
		}
		subnet2Id := awsgo.StringValue(subnet2.SubnetId)
		subnet2VpcId := awsgo.StringValue(subnet2.VpcId)
		if az1 != az2 && subnet1Id != subnet2Id && subnet1VpcId == subnet2VpcId {
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
	require.True(t, len(result.LoadBalancers) > 0, "Could not create test ELBv2")

	balancer := *result.LoadBalancers[0]

	err = svc.WaitUntilLoadBalancerAvailable(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{balancer.LoadBalancerArn},
	})
	require.NoError(t, err)

	return balancer
}

func TestListELBv2(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	elbName := "cloud-nuke-test-" + util.UniqueID()
	balancer := createTestELBv2(t, session, elbName)
	// clean up after this test
	defer nukeAllElbv2Instances(session, []*string{balancer.LoadBalancerArn})

	arns, err := getAllElbv2Instances(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))

	arns, err = getAllElbv2Instances(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))
}

func TestNukeELBv2(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

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

	arns, err := getAllElbv2Instances(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(arns), awsgo.StringValue(balancer.LoadBalancerArn))
}

// Test config file filtering works as expected
func TestShouldIncludeELBv2(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	mockELBv2 := &elbv2.LoadBalancer{
		LoadBalancerName: awsgo.String("cloud-nuke-test"),
		CreatedTime:      awsgo.Time(time.Now()),
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		logging.Logger.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		ELBv2: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	mockIncludeConfig := config.Config{
		ELBv2: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	cases := []struct {
		Name         string
		ELBv2        *elbv2.LoadBalancer
		Config       config.Config
		ExcludeAfter time.Time
		Expected     bool
	}{
		{
			Name:         "ConfigExclude",
			ELBv2:        mockELBv2,
			Config:       mockExcludeConfig,
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Expected:     false,
		},
		{
			Name:         "ConfigInclude",
			ELBv2:        mockELBv2,
			Config:       mockIncludeConfig,
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Expected:     true,
		},
		{
			Name:         "NotOlderThan",
			ELBv2:        mockELBv2,
			Config:       config.Config{},
			ExcludeAfter: time.Now().Add(1 * time.Hour * -1),
			Expected:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeELBv2(c.ELBv2, c.ExcludeAfter, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}
