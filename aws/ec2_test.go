package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/util"
	gruntworkerrors "github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

const (
	ExampleId                   = "a1b2c3d4e5f601345"
	ExampleIdTwo                = "a1b2c3d4e5f654321"
	ExampleIdThree              = "a1b2c3d4e5f632154"
	ExampleVpcId                = "vpc-" + ExampleId
	ExampleVpcIdTwo             = "vpc-" + ExampleIdTwo
	ExampleVpcIdThree           = "vpc-" + ExampleIdThree
	ExampleSubnetId             = "subnet-" + ExampleId
	ExampleSubnetIdTwo          = "subnet-" + ExampleIdTwo
	ExampleSubnetIdThree        = "subnet-" + ExampleIdThree
	ExampleRouteTableId         = "rtb-" + ExampleId
	ExampleNetworkAclId         = "acl-" + ExampleId
	ExampleSecurityGroupId      = "sg-" + ExampleId
	ExampleSecurityGroupIdTwo   = "sg-" + ExampleIdTwo
	ExampleSecurityGroupIdThree = "sg-" + ExampleIdThree
	ExampleInternetGatewayId    = "igw-" + ExampleId
)

func TestListInstances(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	instance, err := CreateTestEC2Instance(session, uniqueTestID, false)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	protectedInstance, err := CreateTestEC2Instance(session, uniqueTestID, true)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	// clean up after this test
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId, protectedInstance.InstanceId}, true)

	instanceIds, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.NotContains(t, instanceIds, instance.InstanceId)
	assert.NotContains(t, instanceIds, protectedInstance.InstanceId)

	instanceIds, err = getAllEc2Instances(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.Contains(t, instanceIds, instance.InstanceId)
	assert.NotContains(t, instanceIds, protectedInstance.InstanceId)

	if err = removeEC2InstanceProtection(ec2.New(session), &protectedInstance); err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
}

func TestNukeInstances(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	_, err = CreateTestEC2Instance(session, uniqueTestID, false)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	instanceIds, err := findEC2InstancesByNameTag(session, uniqueTestID)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	if err := nukeAllEc2Instances(session, instanceIds, true); err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	instances, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour))

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	for _, instanceID := range instanceIds {
		assert.NotContains(t, instances, *instanceID)
	}
}
