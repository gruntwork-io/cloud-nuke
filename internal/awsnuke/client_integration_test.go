package awsnuke_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/internal/awsnuke"
	"github.com/stretchr/testify/assert"
)

var AWSAccountId = "353720269506"

func TestDestroyEC2Instances(t *testing.T) {
	assert := assert.New(t)

	createNewInstance(t, "us-east-1", "ami-6057e21a")

	ctx := context.Background()
	nuke := awsnuke.New(AWSAccountId)
	instances, err := nuke.ListNonProtectedEC2Instances(ctx)
	if assert.NoError(err, "unexpected error: %v", err) {
		return
	}

	err = nuke.DestroyEC2Instances(ctx, instances)
	if assert.NoError(err, "unexpected error: %v", err) {
		return
	}

	// TODO don't use awsnuke in assertion
	instances, err = nuke.ListNonProtectedEC2Instances(ctx)
	if assert.NoError(err, "unexpected error: %v", err) {
		return
	}

	assert.Len(instances, 0, "expected all instances to have been destroyed")
}

func createNewInstance(t *testing.T, region, ami string) {
	ec2client := newEC2Client(region)
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: aws.String("t1.micro"),
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
	}
	reservation, err := ec2client.RunInstances(input)
	if err != nil {
		t.Fatalf("failed to create ec2 instance: %v", err)
	}

	ids := instanceIds(reservation.Instances)
	err = ec2client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: ids,
	})
	if err != nil {
		t.Fatalf("failed while waiting for instances to start running, %v", err)
	}
}

func newEC2Client(region string) *ec2.EC2 {
	sess := session.Must(session.NewSession())
	creds := credentials.NewEnvCredentials()
	return ec2.New(sess, &aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	})
}

func instanceIds(instances []*ec2.Instance) []*string {
	var ids []*string
	for _, instance := range instances {
		ids = append(ids, instance.InstanceId)
	}
	return ids
}
