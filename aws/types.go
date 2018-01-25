package aws

// AwsAccountResources - maps an aws resource to AwsRegionResources
type AwsAccountResources struct {
	Resources map[string][]AwsRegionResources
}

// AwsRegionResource - maps an aws region to a list of resource identifiers
type AwsRegionResources struct {
	RegionName          string
	ResourceIdentifiers []string // Can be list of ec2 instance ids, elb names etc
}
