package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var regions []string
var ec2InstanceIds []*string

// GetAllResources - Lists all aws resources
func GetAllResources() []*string {
	regions = []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1",
		"eu-west-1", "eu-central-1", "eu-west-2", "ap-southeast-1", "ap-southeast-2",
		"ap-northeast-2", "ap-northeast-1", "ap-south-1", "sa-east-1",
	}

	for _, region := range regions {
		session, _ := session.NewSession(&aws.Config{
			Region: aws.String(region)},
		)

		instances, err := getAllEc2Instances(session, region)
		if err == nil {
			ec2InstanceIds = append(ec2InstanceIds, instances...)
		}
	}

	var resources []*string
	resources = append(resources, ec2InstanceIds...)
	return resources
}

// NukeAllResources - Deletes all aws resources
func NukeAllResources() {
	for _, region := range regions {
		session, _ := session.NewSession(&aws.Config{
			Region: aws.String(region)},
		)

		err := nukeAllEc2Instances(session, ec2InstanceIds)
		if err != nil {
			fmt.Println("Could not terminate EC2 instances")
		}
	}
}
