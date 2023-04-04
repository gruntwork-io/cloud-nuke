package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
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
		assert.Failf(t, "Could not create test ELB", errors.WithStackTrace(err).Error())
	}
}

func TestListELBs(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbName := "cloud-nuke-test-" + util.UniqueID()
	createTestELB(t, session, elbName)
	// clean up after this test
	defer nukeAllElbInstances(session, []*string{&elbName})

	elbNames, err := getAllElbInstances(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of ELBs", errors.WithStackTrace(err).Error())
	}

	assert.NotContains(t, awsgo.StringValueSlice(elbNames), elbName)

	elbNames, err = getAllElbInstances(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of ELBs", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(elbNames), elbName)
}

func TestNukeELBs(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := elb.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	elbName := "cloud-nuke-test-" + util.UniqueID()
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
