package resource

import "fmt"

// InvalidResourceTypesSuppliedError is returned when invalid resource types are specified
type InvalidResourceTypesSuppliedError struct {
	InvalidTypes []string
}

func (err InvalidResourceTypesSuppliedError) Error() string {
	return fmt.Sprintf("Invalid resourceTypes %s specified: %s", err.InvalidTypes, "Try --list-resource-types to get a list of valid resource types.")
}

// ResourceTypeAndExcludeFlagsBothPassedError is returned when both --resource-type and --exclude-resource-type are specified
type ResourceTypeAndExcludeFlagsBothPassedError struct{}

func (err ResourceTypeAndExcludeFlagsBothPassedError) Error() string {
	return "You can not specify both --resource-type and --exclude-resource-type"
}
