package main

import (
	"github.com/gruntwork-io/cloud-nuke/commands"
	"github.com/gruntwork-io/gruntwork-cli/entrypoint"
)

// VERSION - Set at build time
var VERSION string

func main() {
	app := commands.CreateCli(VERSION)
	entrypoint.RunApp(app)
}
