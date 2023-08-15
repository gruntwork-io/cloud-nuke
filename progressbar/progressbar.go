package progressbar

import (
	"time"

	"github.com/pterm/pterm"
)

var p *pterm.ProgressbarPrinter

func init() {
	p = &pterm.DefaultProgressbar
	p.RemoveWhenDone = true
	p.ElapsedTimeRoundingFactor = time.Second
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

// StartProgressBarWithLength - Starts the progress bar with the correct number of items
func StartProgressBarWithLength(length int) {
	// Update the progress bar to have the correct width based on the total number of unique resource targteds
	WithTotal(length)
	p := GetProgressbar()
	p.Start()
}
