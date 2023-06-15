package main

import (
	"github.com/gruntwork-io/cloud-nuke/commands"
	"github.com/gruntwork-io/go-commons/entrypoint"
)

// VERSION - Set at build time
var VERSION string

func main() {
	app := commands.CreateCli(VERSION)
	entrypoint.RunApp(app)
}
