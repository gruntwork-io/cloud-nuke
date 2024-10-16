package aws

func ReportGetAllRegisterResources() []AwsResource {
	var resources []AwsResource
	resources = append(resources, getRegisteredGlobalResources()...)
	resources = append(resources, getRegisteredRegionalResources()...)

	return resources
}
