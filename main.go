package main

import (
	"github.com/andrewderr/cloud-nuke-a1/commands"
	"github.com/gruntwork-io/go-commons/entrypoint"
)

// VERSION - Set at build time
var VERSION string

func main() {
	app := commands.CreateCli(VERSION)
	entrypoint.RunApp(app)
}
