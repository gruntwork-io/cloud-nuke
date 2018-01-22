package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/logging"
)

var ec2Instances = make(map[string][]*string)

func buildEntryName(instance ec2.Instance) string {
	return fmt.Sprintf("ec2-%s-%s", *instance.InstanceId, *instance.InstanceType)
}

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

			attr, _ := svc.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
				Attribute:  awsgo.String("disableApiTermination"),
				InstanceId: awsgo.String(instanceID),
			})

			protected := *attr.DisableApiTermination.Value
			if !protected {
				ec2Instances[region] = append(ec2Instances[region], &instanceID)
				entries = append(entries, buildEntryName(*instance))
			}
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

	logging.Logger.Infof("Terminating all EC2 instances in region %s", *session.Config.Region)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}

	_, err := svc.TerminateInstances(params)
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return err
	}

	logging.Logger.Infof("[OK] %d instance(s) terminated in %s", len(instances), *session.Config.Region)
	return nil
}
