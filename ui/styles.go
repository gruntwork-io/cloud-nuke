package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
)

var (
	FailureEmoji           = "❌"
	SuccessEmoji           = "✅"
	FireEmoji              = "🔥"
	TargetEmoji            = "🎯"
	ResourceHighlightStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("202"))
	BoldWarningTextStyle = lipgloss.NewStyle().
				Padding(1).
				Bold(true).
				Italic(true).
				Foreground(lipgloss.Color("202"))
	BlinkingWarningStyle = lipgloss.NewStyle().
				Bold(true).
				Italic(true).
				Blink(true).
				Underline(true).
				Foreground(lipgloss.Color("201"))
)

// WarningMessage is a convenience method to display the supplied string in our pre-configured BoldWarningTextStyle
func WarningMessage(s string) {
	s = strings.ToUpper(s)
	s = BoldWarningTextStyle.Render(s)
	pterm.Warning.Println(s)
	pterm.Println()
}

// UrgentMessage is a convenience method to display the supplied string in our pre-configured BlinkingWarningStyle
// This is appropriate to use as the final prompt to confirm the user really wants to nuke all selected resources
func UrgentMessage(s string) {
	s = strings.ToUpper(s)
	s = BlinkingWarningStyle.Render(s)
	pterm.Error.Prefix.Text = "CRITICAL"
	pterm.Error.Println(s)
	// Replace original prefix to avoid weird edge cases
	pterm.Error.Prefix.Text = "Error"
	pterm.Println()
}
