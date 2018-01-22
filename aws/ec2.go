package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var ec2Instances = make(map[string][]*string)

func getAllEc2Instances(session *session.Session, region string) ([]string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: awsgo.String("instance-state-name"),
				Values: []*string{
					awsgo.String("running"), awsgo.String("pending"),
				},
			},
		},
	}

	output, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, err
	}

	var entries []string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId
			ec2Instances[region] = append(ec2Instances[region], &instanceID)
			entry := fmt.Sprintf("ec2-%s-%s", instanceID, *instance.InstanceType)
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func nukeAllEc2Instances(session *session.Session) error {
	svc := ec2.New(session)
	instances := ec2Instances[*session.Config.Region]

	if len(instances) == 0 {
		return nil
	}

	msg := fmt.Sprintf("Terminating all EC2 instances in region %s...", *session.Config.Region)
	fmt.Print(msg)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}

	_, err := svc.TerminateInstances(params)
	if err != nil {
		fmt.Println("Failed")
		return err
	}

	fmt.Println("Done")
	return nil
}
