package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
)

var (
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

func WarningMessage(s string) {
	s = strings.ToUpper(s)
	s = BoldWarningTextStyle.Render(s)
	pterm.Warning.Println(s)
	pterm.Println()
}

func UrgentMessage(s string) {
	s = strings.ToUpper(s)
	s = BlinkingWarningStyle.Render(s)
	pterm.Error.Prefix.Text = "CRITICAL"
	pterm.Error.Println(s)
	// Replace original prefix to avoid weird edge cases
	pterm.Error.Prefix.Text = "Error"
	pterm.Println()
}
