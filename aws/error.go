package aws

import "fmt"

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
