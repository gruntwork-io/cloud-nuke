package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
)

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
		assert.Failf(t, "Could not create test EC2 instance: %s", err.Error())
	}

	return *runResult.Instances[0]
}

func TestListInstances(t *testing.T) {
	session, _ := session.NewSession(&awsgo.Config{
		Region: awsgo.String("us-west-2")},
	)

	instance := createTestEC2Instance(t, session)
	instanceEntries, err := getAllEc2Instances(session, "us-west-2")

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.NotEqual(t, 0, len(instanceEntries))
	assert.Contains(t, instanceEntries, buildEntryName(instance))
}

func TestNukeInstances(t *testing.T) {
	session, _ := session.NewSession(&awsgo.Config{
		Region: awsgo.String("us-west-2")},
	)

	nukeAllEc2Instances(session)
	instanceEntries, err := getAllEc2Instances(session, "us-west-2")

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	assert.Equal(t, 0, len(instanceEntries))
}
