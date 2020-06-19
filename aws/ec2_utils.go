package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	gruntworkerrors "github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAMIIdByName(svc *ec2.EC2, name string) (string, error) {
	imagesResult, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{awsgo.String("self"), awsgo.String("amazon")},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("name"),
				Values: []*string{awsgo.String(name)},
			},
		},
	})

	if err != nil {
		return "", gruntworkerrors.WithStackTrace(err)
	}

	if len(imagesResult.Images) == 0 {
		return "", gruntworkerrors.WithStackTrace(fmt.Errorf("No images found with name %s", name))
	}

	image := imagesResult.Images[0]
	return awsgo.StringValue(image.ImageId), nil
}

// runAndWaitForInstance - Given a preconstructed ec2.RunInstancesInput object,
// make the API call to run the instance and then wait for the instance to be
// up and running before returning.
func runAndWaitForInstance(svc *ec2.EC2, name string, params *ec2.RunInstancesInput) (ec2.Instance, error) {
	runResult, err := svc.RunInstances(params)
	if err != nil {
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	if len(runResult.Instances) == 0 {
		err := fmt.Errorf("Could not create test EC2 instance")
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
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
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
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
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
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
		return ec2.Instance{}, gruntworkerrors.WithStackTrace(err)
	}

	return *runResult.Instances[0], nil

}

// CreateTestEC2Instance - creates a test EC2 instance
func CreateTestEC2Instance(session *session.Session, name string, protected bool) (ec2.Instance, error) {
	svc := ec2.New(session)
	var instance ec2.Instance

	imageID, err := getAMIIdByName(svc, "amzn-ami-hvm-2018.03.0.20190826-x86_64-gp2")
	if err != nil {
		return instance, err
	}

	params := &ec2.RunInstancesInput{
		ImageId:               awsgo.String(imageID),
		InstanceType:          awsgo.String("t3.micro"),
		MinCount:              awsgo.Int64(1),
		MaxCount:              awsgo.Int64(1),
		DisableApiTermination: awsgo.Bool(protected),
	}
	instance, err = runAndWaitForInstance(svc, name, params)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

func removeEC2InstanceProtection(svc *ec2.EC2, instance *ec2.Instance) error {
	// make instance unprotected so it can be cleaned up
	_, err := svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: awsgo.Bool(false),
		},
		InstanceId: instance.InstanceId,
	})

	return err
}

func findEC2InstancesByNameTag(session *session.Session, name string) ([]*string, error) {
	var instanceIds []*string
	output, err := ec2.New(session).DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		return instanceIds, err
	}

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

	return instanceIds, nil
}
