package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/gruntwork-io/aws-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestELB(t *testing.T, session *session.Session, name string) {
	svc := elb.New(session)

	param := &elb.CreateLoadBalancerInput{
		AvailabilityZones: []*string{
			awsgo.String(awsgo.StringValue(session.Config.Region) + "a"),
		},
		LoadBalancerName: awsgo.String(name),
		Listeners: []*elb.Listener{
			&elb.Listener{
				InstancePort:     awsgo.Int64(80),
				LoadBalancerPort: awsgo.Int64(80),
				Protocol:         awsgo.String("HTTP"),
			},
		},
	}

	_, err := svc.CreateLoadBalancer(param)
	if err != nil {
		assert.Fail(t, "Could not create test ELB: %v", err)
	}
}

func TestListELBs(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbName := "aws-nuke-test-" + util.UniqueID()
	createTestELB(t, session, elbName)
	// clean up after this test
	defer nukeAllElbInstances(session, []*string{&elbName})

	elbNames, err := getAllElbInstances(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ELBs: %v", err)
	}

	assert.NotContains(t, awsgo.StringValueSlice(elbNames), elbName)

	elbNames, err = getAllElbInstances(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ELBs: %v", err)
	}

	assert.Contains(t, awsgo.StringValueSlice(elbNames), elbName)
}

func TestNukeELBs(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := elb.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbName := "aws-nuke-test-" + util.UniqueID()
	createTestELB(t, session, elbName)

	_, err = svc.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{
			awsgo.String(elbName),
		},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllElbInstances(session, []*string{&elbName}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbNames, err := getAllElbInstances(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of ELBs: %v", err)
	}

	assert.NotContains(t, awsgo.StringValueSlice(elbNames), elbName)
}
