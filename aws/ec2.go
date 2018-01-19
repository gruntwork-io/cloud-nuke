package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func getAllEc2Instances(session *session.Session, region string) ([]*string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"), aws.String("pending"),
				},
			},
		},
	}

	output, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, err
	}

	var instanceIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId
			instanceIds = append(instanceIds, &instanceID)
		}
	}

	return instanceIds, nil
}

func nukeAllEc2Instances(session *session.Session, instanceIds []*string) error {
	svc := ec2.New(session)
	params := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}

	_, err := svc.TerminateInstances(params)
	if err != nil {
		return err
	}

	return nil
}
