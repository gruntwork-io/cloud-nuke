package aws

import (
	"errors"
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
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
	ExampleSecurityGroupRuleId  = "sgr-" + ExampleId
	ExampleInternetGatewayId    = "igw-" + ExampleId
	ExampleEndpointId           = "vpce-" + ExampleId
)

// getAMIIdByName - Retrieves an AMI ImageId given the name of the Id. Used for
// retrieving a standard AMI across AWS regions.
func getAMIIdByName(svc *ec2.EC2, name string) (string, error) {
	imagesResult, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{awsgo.String("self"), awsgo.String("amazon")},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("name"),
				Values: []*string{awsgo.String(name)},
			},
		},
	})

	if err != nil {
		return "", gruntworkerrors.WithStackTrace(err)
	}

	if len(imagesResult.Images) == 0 {
		return "", gruntworkerrors.WithStackTrace(fmt.Errorf("No images found with name %s", name))
	}

	image := imagesResult.Images[0]
	return awsgo.StringValue(image.ImageId), nil
}

// runAndWaitForInstance - Given a preconstructed ec2.RunInstancesInput object,
// make the API call to run the instance and then wait for the instance to be
// up and running before returning.
func runAndWaitForInstance(svc *ec2.EC2, name string, params *ec2.RunInstancesInput) (ec2.Instance, error) {
	runResult, err := svc.RunInstances(params)
	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	if len(runResult.Instances) == 0 {
		err := errors.New("Could not create test EC2Instance instance")
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	err = svc.WaitUntilInstanceExists(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("instance-id"),
				Values: []*string{runResult.Instances[0].InstanceId},
			},
		},
	})

	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	// Add test tag to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String(name),
			},
		},
	})

	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	// EC2Instance Instance must be in a running before this function returns
	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("instance-id"),
				Values: []*string{runResult.Instances[0].InstanceId},
			},
		},
	})

	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	return *runResult.Instances[0], nil

}

func createTestEC2Instance(t *testing.T, session *session.Session, name string, protected bool) ec2.Instance {
	svc := ec2.New(session)

	imageID, err := getAMIIdByName(svc, "amzn-ami-hvm-2018.03.0.20220315.0-x86_64-gp2")
	if err != nil {
		assert.Fail(t, err.Error())
	}

	params := &ec2.RunInstancesInput{
		ImageId:               awsgo.String(imageID),
		InstanceType:          awsgo.String("t3.micro"),
		MinCount:              awsgo.Int64(1),
		MaxCount:              awsgo.Int64(1),
		DisableApiTermination: awsgo.Bool(protected),
	}
	instance, err := runAndWaitForInstance(svc, name, params)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	return instance
}

func removeEC2InstanceProtection(svc *ec2.EC2, instance *ec2.Instance) error {
	// make instance unprotected so it can be cleaned up
	_, err := svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: awsgo.Bool(false),
		},
		InstanceId: instance.InstanceId,
	})

	return err
}

func findEC2InstancesByNameTag(t *testing.T, session *session.Session, name string) []*string {
	output, err := ec2.New(session).DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	var instanceIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			// Retrive only IDs of instances with the unique test tag
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					if *tag.Value == name {
						instanceIds = append(instanceIds, &instanceID)
					}
				}
			}

		}
	}

	return instanceIds
}

func TestListInstances(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	instance := createTestEC2Instance(t, session, uniqueTestID, false)
	protectedInstance := createTestEC2Instance(t, session, uniqueTestID, true)
	// clean up after this test
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId, protectedInstance.InstanceId})

	instanceIds, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2Instance Instances")
	}

	assert.NotContains(t, instanceIds, instance.InstanceId)
	assert.NotContains(t, instanceIds, protectedInstance.InstanceId)

	instanceIds, err = getAllEc2Instances(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2Instance Instances")
	}

	assert.Contains(t, instanceIds, instance.InstanceId)
	assert.NotContains(t, instanceIds, protectedInstance.InstanceId)

	if err = removeEC2InstanceProtection(ec2.New(session), &protectedInstance); err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
}

func TestNukeInstances(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	createTestEC2Instance(t, session, uniqueTestID, false)

	instanceIds := findEC2InstancesByNameTag(t, session, uniqueTestID)

	if err := nukeAllEc2Instances(session, instanceIds); err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	instances, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour), config.Config{})

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2Instance Instances")
	}

	for _, instanceID := range instanceIds {
		assert.NotContains(t, instances, *instanceID)
	}
}

func TestGetEC2ResourceNameTagValue(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	cases := []struct {
		Name          string
		Tags          []*ec2.Tag
		Expected      string
		ExpectedError error
	}{
		{
			Name: "HasName",
			Tags: []*ec2.Tag{
				{
					Key:   awsgo.String("Name"),
					Value: awsgo.String("cloud-nuke-test"),
				},
				{
					Key:   awsgo.String("Foo"),
					Value: awsgo.String("Bar"),
				},
			},
			Expected:      "cloud-nuke-test",
			ExpectedError: nil,
		},
		{
			Name: "MissingName",
			Tags: []*ec2.Tag{
				{
					Key:   awsgo.String("foo"),
					Value: awsgo.String("bar"),
				},
			},
			Expected: "",
		},
		{
			Name:     "NoTags",
			Tags:     []*ec2.Tag{},
			Expected: "",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result, err := GetEC2ResourceNameTagValue(c.Tags)
			assert.Equal(t, c.Expected, result)
			switch err {
			case nil:
				assert.Equal(t, c.ExpectedError, err)
			default:
				assert.Error(t, err)
			}
		})
	}
}

// Test config file filtering works as expected
func TestShouldIncludeInstanceId(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")

	mockInstance := &ec2.Instance{
		LaunchTime: awsgo.Time(time.Now()),
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String("cloud-nuke-test"),
			},
			{
				Key:   awsgo.String("Foo"),
				Value: awsgo.String("Bar"),
			},
		},
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		logging.Logger.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		EC2Instance: config.ResourceType{
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
		EC2Instance: config.ResourceType{
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
		Instance     *ec2.Instance
		Config       config.Config
		ExcludeAfter time.Time
		Protected    bool
		Expected     bool
	}{
		{
			Name:         "ConfigExclude",
			Instance:     mockInstance,
			Config:       mockExcludeConfig,
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Protected:    false,
			Expected:     false,
		},
		{
			Name:         "ConfigInclude",
			Instance:     mockInstance,
			Config:       mockIncludeConfig,
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Protected:    false,
			Expected:     true,
		},
		{
			Name:         "NotOlderThan",
			Instance:     mockInstance,
			Config:       config.Config{},
			ExcludeAfter: time.Now().Add(1 * time.Hour * -1),
			Protected:    false,
			Expected:     false,
		},
		{
			Name:         "Protected",
			Instance:     mockInstance,
			Config:       config.Config{},
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Protected:    true,
			Expected:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeInstanceId(c.Instance, c.ExcludeAfter, c.Protected, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}
