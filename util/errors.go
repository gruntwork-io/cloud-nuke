package util

import (
	"fmt"
	"strings"
)

// MultiErr is a meta error that can be used to track multiple errors in a routine (e.g., for loop) so that you can
// return all the errors in aggregate.
type MultiErr struct {
	errors []error
}

func (err *MultiErr) IsEmpty() bool {
	return len(err.errors) == 0
}

// NOTE: this must be a pointer method so that the errors list is updated on the same object
func (err *MultiErr) Add(newErr error) {
	if err.errors == nil {
		err.errors = []error{}
	}
	err.errors = append(err.errors, newErr)
}

func (err *MultiErr) Error() string {
	out := []string{}
	for _, childErr := range err.errors {
		out = append(out, childErr.Error())
	}
	return fmt.Sprintf("Encountered multiple errors:\n%s", strings.Join(out, "\n"))
}
