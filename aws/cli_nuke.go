package aws

func NukeResourcesViaCLI(a *AwsAccountResources, regions []string) error {
	// Log which resource types will be inspected
	// NOTE: The inspect functionality currently does not support config file, so we short circuit the logic with an empty struct.
	return NukeAllResources(a, regions)
}
