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

type UnsupportedProviderError struct {
	Name string
}

func (e UnsupportedProviderError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("Invalid cloud provider specified. Possible values aws | azure | gcp")
	}

	return fmt.Sprintf("%s is not currently supported by cloud-nuke", e.Name)
}
