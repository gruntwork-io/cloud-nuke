package commands

import (
	"strings"

	"github.com/fatih/color"
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
	app.Usage = "A CLI tool to cleanup AWS resources (ASG, ELB, ELBv2, EBS, EC2). THIS TOOL WILL COMPLETELY REMOVE ALL RESOURCES AND ITS EFFECTS ARE IRREVERSIBLE!!!"
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "exclude-region",
			Usage: "regions to exclude",
		},
	}
	app.Action = errors.WithPanicHandling(awsNuke)

	return app
}

// Nuke it all!!!
func awsNuke(c *cli.Context) error {
	logging.Logger.Infoln("Retrieving all active AWS resources")

	account, err := aws.GetAllResources(c.StringSlice("exclude"))

	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(account.Resources) == 0 {
		logging.Logger.Infoln("Nothing to nuke, you're all good!")
		return nil
	}

	for region, resourcesInRegion := range account.Resources {
		for _, resources := range resourcesInRegion.Resources {
			for _, identifier := range resources.ResourceIdentifiers() {
				logging.Logger.Infof("%s-%s-%s", resources.ResourceName(), identifier, region)
			}
		}
	}

	color := color.New(color.FgHiRed, color.Bold)
	color.Println("\nTHE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

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
