package aws

import (
	"fmt"

	"github.com/gruntwork-io/cloud-nuke/util"
)

type CouldNotSelectRegionError struct {
	Underlying error
}

func (err CouldNotSelectRegionError) Error() string {
	return fmt.Sprintf("Unable to determine target region set. Please double check your combination of target and excluded regions. Original error: %v", err.Underlying)
}

type CouldNotDetermineEnabledRegionsError struct {
	Underlying error
}

func (err CouldNotDetermineEnabledRegionsError) Error() string {
	return fmt.Sprintf("Unable to determine enabled regions in target account. Original error: %v", err.Underlying)
}

// InvalidResourceTypesSuppliedError is returned when invalid resource type names are provided.
type InvalidResourceTypesSuppliedError = util.InvalidResourceTypesSuppliedError

// ResourceTypeAndExcludeFlagsBothPassedError is returned when both --resource-type and --exclude-resource-type are specified.
type ResourceTypeAndExcludeFlagsBothPassedError = util.ResourceTypeAndExcludeFlagsBothPassedError

type InvalidTimeStringPassedError struct {
	Entry      string
	Underlying error
}

func (err InvalidTimeStringPassedError) Error() string {
	return fmt.Sprintf("Could not parse %s as a valid time duration. Underlying error: %s", err.Entry, err.Underlying)
}

type QueryCreationError struct {
	Underlying error
}

func (err QueryCreationError) Error() string {
	return fmt.Sprintf("Error forming a cloud-nuke Query with supplied parameters. Original error: %v", err.Underlying)
}

type ResourceInspectionError struct {
	Underlying error
}

func (err ResourceInspectionError) Error() string {
	return fmt.Sprintf("Error encountered when querying for account resources. Original error: %v", err.Underlying)
}
