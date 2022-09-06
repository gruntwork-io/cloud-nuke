package spinner

import "github.com/pterm/pterm"

var s pterm.SpinnerPrinter = pterm.DefaultSpinner

func init() {
	s.Sequence = []string{"☢️ ", "💥", "🔥", "❌"}
}

func GetSpinner() pterm.SpinnerPrinter {
	return s
}

func UpdateText(t string) {
	s.UpdateText(t)
}
