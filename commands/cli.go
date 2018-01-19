package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
		fmt.Println("Retrieving all active AWS resources...")
		resources := aws.GetAllResources()

		if len(resources) == 0 {
			fmt.Println("No resources to nuke, you're all good!")
			return nil
		}

		fmt.Println("The following resources will be deleted: ")
		for _, resource := range resources {
			fmt.Println(*resource)
		}

		fmt.Print("\nAre you sure you want to nuke all listed resources (Y/N)? ")

		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(input) == "y\n" {
			fmt.Println("Deleting all resources...")
			aws.NukeAllResources()
		}

		return nil
	}

	return app
}
