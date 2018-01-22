package commands

import (
	"fmt"

	"github.com/gruntwork-io/aws-nuke/aws"
	"github.com/gruntwork-io/gruntwork-cli/shell"
	"github.com/urfave/cli"
)

// CreateCli - Create the CLI app with all commands, flags, and usage text configured.
func CreateCli(version string) *cli.App {
	app := cli.NewApp()

	app.Name = "aws-nuke"
	app.HelpName = app.Name
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.Version = version
	app.Usage = "A CLI tool to cleanup AWS resources"
	app.Action = func(c *cli.Context) error {
		fmt.Println("Retrieving all active AWS resources...")
		resources := aws.GetAllResources()

		if len(resources) == 0 {
			fmt.Println("Nothing to nuke, you're all good!")
			return nil
		}

		fmt.Println("\nThe following resources will be nuked: ")
		for _, resource := range resources {
			fmt.Println(resource)
		}

		prompt := "\nAre you sure you want to nuke all listed resources"
		proceed, err := shell.PromptUserForYesNo(prompt, shell.NewShellOptions())

		if err != nil {
			return err
		}

		if proceed {
			aws.NukeAllResources()
		}

		return nil
	}

	return app
}
