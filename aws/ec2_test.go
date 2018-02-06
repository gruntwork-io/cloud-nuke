package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func getLinuxAmiIDForRegion(region string) string {
	switch region {
	case "us-east-1":
		return "ami-97785bed"
	case "us-east-2":
		return "ami-f63b1193"
	case "us-west-1":
		return "ami-824c4ee2"
	case "us-west-2":
		return "ami-f2d3638a"
	case "ca-central-1":
		return "ami-a954d1cd"
	case "eu-west-1":
		return "ami-d834aba1"
	case "eu-west-2":
		return "ami-403e2524"
	case "eu-west-3":
		return "ami-8ee056f3"
	case "eu-central-1":
		return "ami-5652ce39"
	case "ap-northeast-1":
		return "ami-ceafcba8"
	case "ap-northeast-2":
		return "ami-863090e8"
	case "ap-south-1":
		return "ami-531a4c3c"
	case "ap-southeast-1":
		return "ami-68097514"
	case "ap-southeast-2":
		return "ami-942dd1f6"
	case "sa-east-1":
		return "ami-84175ae8"
	default:
		return ""
	}
}

func createTestEC2Instance(t *testing.T, session *session.Session, name string) ec2.Instance {
	svc := ec2.New(session)

	params := &ec2.RunInstancesInput{
		ImageId:      awsgo.String(getLinuxAmiIDForRegion(*session.Config.Region)),
		InstanceType: awsgo.String("t1.micro"),
		MinCount:     awsgo.Int64(1),
		MaxCount:     awsgo.Int64(1),
	}

	runResult, err := svc.RunInstances(params)
	if err != nil {
		assert.Fail(t, "Could not create test EC2 instance")
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
		assert.Fail(t, errors.WithStackTrace(err).Error())
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
		assert.Failf(t, "Could not tag EC2 instance: %s", errors.WithStackTrace(err).Error())
	}

	// EC2 Instance must be in a running before this function returns
	err = ec2.New(session).WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("instance-id"),
				Values: []*string{runResult.Instances[0].InstanceId},
			},
		},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return *runResult.Instances[0]
}

func findEC2InstancesByNameTag(output *ec2.DescribeInstancesOutput, name string) []*string {
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
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	instance := createTestEC2Instance(t, session, uniqueTestID)
	instanceIds, err := getAllEc2Instances(session, region)

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.Contains(t, instanceIds, instance.InstanceId)

	// clean up after this test
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId})
}

func TestNukeInstances(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	createTestEC2Instance(t, session, uniqueTestID)

	output, err := ec2.New(session).DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	instanceIds := findEC2InstancesByNameTag(output, uniqueTestID)

	if err := nukeAllEc2Instances(session, instanceIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	instances, err := getAllEc2Instances(session, region)

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	for _, instanceID := range instanceIds {
		assert.NotContains(t, instances, *instanceID)
	}
}
