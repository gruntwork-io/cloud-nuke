package commands

import (
	"strings"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
)

// renderNukeConfirmationPrompt displays a confirmation prompt before nuking resources.
// Returns true if the user confirms, false otherwise.
func renderNukeConfirmationPrompt(prompt string, numRetryCount int) (bool, error) {
	prompts := 0

	pterm.Println()
	pterm.Warning.Println("THE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

	for prompts < numRetryCount {
		confirmPrompt := pterm.DefaultInteractiveTextInput.WithMultiLine(false)
		input, err := confirmPrompt.Show(prompt)
		if err != nil {
			logging.Errorf("[Failed to render prompt] %s", err)
			return false, errors.WithStackTrace(err)
		}

		response := strings.ToLower(strings.TrimSpace(input))
		if response == "nuke" {
			pterm.Println()
			return true, nil
		}

		pterm.Println()
		pterm.Error.Printfln("Invalid value was entered: %s. Try again.", input)
		prompts++
	}

	pterm.Println()
	return false, nil
}
