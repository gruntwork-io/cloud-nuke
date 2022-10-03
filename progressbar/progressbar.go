package progressbar

import "github.com/pterm/pterm"

var p *pterm.ProgressbarPrinter

func init() {
	p = &pterm.DefaultProgressbar
	p.RemoveWhenDone = true
}

func GetProgressbar() *pterm.ProgressbarPrinter {
	return p
}

func WithTotal(i int) {
	p = p.WithTotal(i)
}

func UpdateTitle(t string) {
	p = p.UpdateTitle(t)
}
