package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Build string that'll be shown in resources list
func buildEntryName(instance ec2.Instance) string {
	return fmt.Sprintf(
		"ec2-%s-%s",
		awsgo.StringValue(instance.InstanceId),
		awsgo.StringValue(instance.InstanceType),
	)
}

// Returns a formatted string of EC2 instance ids
func getAllEc2Instances(session *session.Session, region string) ([]*string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: awsgo.String("instance-state-name"),
				Values: []*string{
					awsgo.String("running"), awsgo.String("pending"),
					awsgo.String("stopped"), awsgo.String("stopping"),
				},
			},
		},
	}

	output, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var instancesIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			attr, err := svc.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
				Attribute:  awsgo.String("disableApiTermination"),
				InstanceId: awsgo.String(instanceID),
			})

			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			protected := *attr.DisableApiTermination.Value
			// Exclude protected EC2 instances
			if !protected {
				instancesIds = append(instancesIds, &instanceID)
			}
		}
	}

	return instancesIds, nil
}

// Deletes all non protected EC2 instances
func nukeAllEc2Instances(session *session.Session, instancesIds []*string) error {
	svc := ec2.New(session)

	if len(instancesIds) == 0 {
		logging.Logger.Infof("No EC2 instances to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Terminating all EC2 instances in region %s", *session.Config.Region)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: instancesIds,
	}

	_, err := svc.TerminateInstances(params)
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Logger.Infof("[OK] %d instance(s) terminated in %s", len(instancesIds), *session.Config.Region)
	return nil
}
