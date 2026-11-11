package main

import (
	"os"
	"runtime/debug"

	"github.com/gruntwork-io/cloud-nuke/commands"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// VERSION - Set at build time
var VERSION string

func main() {
	app := commands.CreateCli(VERSION)

	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	log := logger.WithField("binary", app.Name).WithField("version", app.Version)

	// Stop urfave/cli from exiting on its own so every error is logged below and exits 1.
	cli.OsExiter = func(int) {}

	defer func() {
		if r := recover(); r != nil {
			log.Error(r)
			if os.Getenv("GRUNTWORK_DEBUG") != "" {
				log.Error(string(debug.Stack()))
			}
			os.Exit(1)
		}
	}()

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
