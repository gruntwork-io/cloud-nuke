package commands

import (
	"fmt"
)

type InvalidFlagError struct {
	Name  string
	Value string
}

func (e InvalidFlagError) Error() string {
	return fmt.Sprintf("Invalid value %s for flag %s", e.Value, e.Name)
}

type ConfigFileReadError struct {
	FilePath   string
	Underlying error
}

func (e ConfigFileReadError) Error() string {
	return fmt.Sprintf("Error reading config file %s: %v", e.FilePath, e.Underlying)
}

type InvalidDurationError struct {
	FlagName   string
	Value      string
	Underlying error
}

func (e InvalidDurationError) Error() string {
	return fmt.Sprintf("Invalid duration value '%s' for flag --%s: %v", e.Value, e.FlagName, e.Underlying)
}

type InvalidLogLevelError struct {
	Value      string
	Underlying error
}

func (e InvalidLogLevelError) Error() string {
	return fmt.Sprintf("Invalid log level '%s': %v", e.Value, e.Underlying)
}
