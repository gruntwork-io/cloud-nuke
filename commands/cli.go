package commands

import (
	"github.com/gruntwork-io/aws-nuke/aws"
	"github.com/gruntwork-io/aws-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
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
	app.Action = errors.WithPanicHandling(awsNuke)

	return app
}

// Nuke it all!!!
func awsNuke(c *cli.Context) error {
	logging.Logger.Infoln("Retrieving all active AWS resources")

	resources := aws.GetAllResources()
	if len(resources) == 0 {
		logging.Logger.Infoln("Nothing to nuke, you're all good!")
		return nil
	}

	for _, resource := range resources {
		logging.Logger.Infoln(resource)
	}

	prompt := "\nAre you sure you want to nuke all listed resources"
	shellOptions := shell.ShellOptions{Logger: logging.Logger}
	proceed, err := shell.PromptUserForYesNo(prompt, &shellOptions)

	if err != nil {
		return err
	}

	if proceed {
		aws.NukeAllResources()
	}

	return nil
}
