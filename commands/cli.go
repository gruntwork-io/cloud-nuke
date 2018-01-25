package commands

import (
	"strings"

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
	app.Usage = "A CLI tool to cleanup AWS resources (EC2). THIS TOOL WILL COMPLETELY REMOVE ALL RESOURCES AND ITS EFFECTS ARE IRREVERSIBLE!!!"
	app.Action = errors.WithPanicHandling(awsNuke)

	return app
}

// Nuke it all!!!
func awsNuke(c *cli.Context) error {
	logging.Logger.Infoln("Retrieving all active AWS resources")

	account, err := aws.GetAllResources()

	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(account.Resources) == 0 {
		logging.Logger.Infoln("Nothing to nuke, you're all good!")
		return nil
	}

	for resource, regionResources := range account.Resources {
		for _, region := range regionResources {
			for _, identifier := range region.ResourceIdentifiers {
				logging.Logger.Infof("%s-%s-%s", resource, identifier, region.RegionName)
			}
		}
	}

	prompt := "\nAre you sure you want to nuke all listed resources? Enter 'nuke' to confirm: "
	shellOptions := shell.ShellOptions{Logger: logging.Logger}
	input, err := shell.PromptUserForInput(prompt, &shellOptions)

	if err != nil {
		return errors.WithStackTrace(err)
	}

	if strings.ToLower(input) == "nuke" {
		aws.NukeAllResources(account)
	}

	return nil
}
