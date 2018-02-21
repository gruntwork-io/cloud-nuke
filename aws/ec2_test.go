package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestEC2Instance(t *testing.T, session *session.Session, name string, protected bool) ec2.Instance {
	svc := ec2.New(session)

	imagesResult, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{awsgo.String("self"), awsgo.String("amazon")},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("name"),
				Values: []*string{awsgo.String("amzn-ami-hvm-2017.09.1.20180115-x86_64-gp2")},
			},
		},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	imageID := *imagesResult.Images[0].ImageId

	params := &ec2.RunInstancesInput{
		ImageId:               awsgo.String(imageID),
		InstanceType:          awsgo.String("t2.micro"),
		MinCount:              awsgo.Int64(1),
		MaxCount:              awsgo.Int64(1),
		DisableApiTermination: awsgo.Bool(protected),
	}

	runResult, err := svc.RunInstances(params)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if len(runResult.Instances) == 0 {
		assert.Fail(t, "Could not create test EC2 instance in "+*session.Config.Region)
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
	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
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

func removeEC2InstanceProtection(svc *ec2.EC2, instance *ec2.Instance) {
	// make instance unprotected so it can be cleaned up
	svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: awsgo.Bool(false),
		},
		InstanceId: instance.InstanceId,
	})
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
	instance := createTestEC2Instance(t, session, uniqueTestID, false)
	protectedInstance := createTestEC2Instance(t, session, uniqueTestID, true)
	// clean up after this test
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId, protectedInstance.InstanceId})

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

	removeEC2InstanceProtection(ec2.New(session), &protectedInstance)
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
	createTestEC2Instance(t, session, uniqueTestID, false)

	output, err := ec2.New(session).DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	instanceIds := findEC2InstancesByNameTag(output, uniqueTestID)

	if err := nukeAllEc2Instances(session, instanceIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	instances, err := getAllEc2Instances(session, region, time.Now().Add(1*time.Hour))

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	for _, instanceID := range instanceIds {
		assert.NotContains(t, instances, *instanceID)
	}
}
