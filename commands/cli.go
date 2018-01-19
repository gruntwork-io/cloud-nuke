package commands

import (
	"fmt"

	"github.com/gruntwork-io/aws-nuke/aws"
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
		fmt.Println("The following resources will be deleted: ")

		resources := aws.GetAllResources()
		for _, resource := range resources {
			fmt.Println(resource)
		}

		return nil
	}

	return app
}
