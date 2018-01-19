package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func getAllEc2Instances(region string) ([]string, error) {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	svc := ec2.New(session)
	output, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	var instanceIds []string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceIds = append(instanceIds, *instance.InstanceId)
		}
	}

	return instanceIds, nil
}
