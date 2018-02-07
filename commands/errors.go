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
