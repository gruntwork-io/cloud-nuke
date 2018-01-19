package aws

var resources []string
var ec2Instances []string

// GetAllResources - Lists all aws resources
func GetAllResources() []string {
	regions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1",
		"eu-west-1", "eu-central-1", "eu-west-2", "ap-southeast-1", "ap-southeast-2",
		"ap-northeast-2", "ap-northeast-1", "ap-south-1", "sa-east-1",
	}

	for _, region := range regions {
		instances, err := getAllEc2Instances(region)
		if err == nil {
			ec2Instances = append(ec2Instances, instances...)
		}
	}

	resources = append(resources, ec2Instances...)
	return resources
}
