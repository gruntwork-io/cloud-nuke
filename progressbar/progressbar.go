package progressbar

import "github.com/pterm/pterm"

var p pterm.ProgressbarPrinter = pterm.DefaultProgressbar

func GetProgressbar() *pterm.ProgressbarPrinter {
	return &p
}

func UpdateTitle(t string) {
	p.UpdateTitle(t)
}
