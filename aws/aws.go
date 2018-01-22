package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var regions []string

// GetAllResources - Lists all aws resources
func GetAllResources() []string {
	regions = []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1",
		"eu-west-1", "eu-central-1", "eu-west-2", "ap-southeast-1", "ap-southeast-2",
		"ap-northeast-2", "ap-northeast-1", "ap-south-1", "sa-east-1",
	}

	var resources []string
	for _, region := range regions {
		session, _ := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		instances, err := getAllEc2Instances(session, region)
		if err == nil {
			resources = append(resources, instances...)
		}
	}

	return resources
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources() {
	for _, region := range regions {
		session, _ := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		nukeAllEc2Instances(session)
	}
}
