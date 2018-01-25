package aws

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

var uniqueTestID = "aws-nuke-test-" + uniqueID()

// Returns a unique (ish) id we can attach to resources and tfstate files so they don't conflict with each other
// Uses base 62 to generate a 6 character string that's unlikely to collide with the handful of tests we run in
// parallel. Based on code here: http://stackoverflow.com/a/9543797/483528
func uniqueID() string {

	const BASE_62_CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const UNIQUE_ID_LENGTH = 6 // Should be good for 62^6 = 56+ billion combinations

	var out bytes.Buffer

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < UNIQUE_ID_LENGTH; i++ {
		out.WriteByte(BASE_62_CHARS[rand.Intn(len(BASE_62_CHARS))])
	}

	return out.String()
}

func createTestEC2Instance(t *testing.T, session *session.Session) ec2.Instance {
	svc := ec2.New(session)

	params := &ec2.RunInstancesInput{
		ImageId:      awsgo.String("ami-e7527ed7"),
		InstanceType: awsgo.String("t1.micro"),
		MinCount:     awsgo.Int64(1),
		MaxCount:     awsgo.Int64(1),
	}

	runResult, err := svc.RunInstances(params)
	if err != nil {
		assert.Failf(t, "Could not create test EC2 instance: %s", errors.WithStackTrace(err).Error())
	}

	// Add test tag to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String(uniqueTestID),
			},
		},
	})

	if err != nil {
		assert.Failf(t, "Could not tag EC2 instance: %s", errors.WithStackTrace(err).Error())
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
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String("us-west-2")},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	instance := createTestEC2Instance(t, session)
	instanceIds, err := getAllEc2Instances(session, "us-west-2")

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.Contains(t, instanceIds, instance.InstanceId)
}

func TestNukeInstances(t *testing.T) {
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String("us-west-2")},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	output, err := ec2.New(session).DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	instanceIds := findEC2InstancesByNameTag(output, uniqueTestID)

	if err := nukeAllEc2Instances(session, instanceIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	instances, err := getAllEc2Instances(session, "us-west-2")

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	for _, instanceID := range instanceIds {
		assert.NotContains(t, instances, *instanceID)
	}
}
