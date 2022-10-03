package spinner

import (
	"github.com/pterm/pterm"
)

var s *pterm.SpinnerPrinter

func init() {
	s = &pterm.SpinnerPrinter{}
}

func GetSpinner() *pterm.SpinnerPrinter {
	return s
}
