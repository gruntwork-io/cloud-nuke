package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gruntwork-io/aws-nuke/internal/awsnuke"
	"github.com/gruntwork-io/gruntwork-cli/entrypoint"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"github.com/gruntwork-io/gruntwork-cli/shell"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var VERSION string

func main() {
	app := cli.NewApp()
	app.Name = "aws-nuke"
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.Version = VERSION

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "non-interactive",
			Usage: "Assume yes for all interactive prompts",
		},
	}

	app.Action = NewNukeAction().ActionHandler

	entrypoint.RunApp(app)
}

type NukeAction struct {
	log *logrus.Logger
}

func NewNukeAction() *NukeAction {
	return &NukeAction{
		log: logging.GetLogger("aws-nuke"),
	}
}

func (a *NukeAction) ActionHandler(c *cli.Context) error {
	if c.NArg() < 1 {
		return cli.NewExitError("insufficient number of arguments", 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	id := c.Args().Get(0)
	nuke := awsnuke.New(id)
	ec2Instances, err := nuke.ListNonProtectedEC2Instances(ctx)
	if err != nil {
		a.log.Debugf("%v", err)
	}

	displaySummary(ec2Instances, id, a.log)

	if !userConfirmation(c.Bool("non-interactive"), a.log) {
		a.log.Info("exiting without deleting resources")
		return nil
	}

	err = nuke.DestroyEC2Instances(ctx, ec2Instances)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("failed to delete ec2 instances, %v", err), 2)
	}

	return nil
}

func displaySummary(instances []awsnuke.EC2Instance, id string, log *logrus.Logger) {
	for _, instance := range instances {
		log.Infof("ec2 instance: %s", instance.Id)
	}

	log.Infof("resources to be deleted for aws account '%s'", id)
	log.Infof("instances: %d", len(instances))
	log.Warn("destroying these resources will be perminate and may not be undone")
}

func userConfirmation(nonInteractive bool, log *logrus.Logger) bool {
	opts := shell.NewShellOptions()
	opts.NonInteractive = nonInteractive
	opts.Logger = log
	answer, err := shell.PromptUserForYesNo("nuke resources?", opts)
	if err != nil {
		log.Error("error retrieving user input")
		return false
	}
	return answer
}
