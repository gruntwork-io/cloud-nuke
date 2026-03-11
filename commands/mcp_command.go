package commands

import (
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/mcp"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/urfave/cli/v2"
)

// MCP flag names
const (
	FlagMCPReadOnly             = "read-only"
	FlagMCPAllowedRegions       = "allowed-regions"
	FlagMCPAllowedResourceTypes = "allowed-resource-types"
	FlagMCPAllowedProjects      = "allowed-projects"
	FlagMCPMaxResourcesPerNuke  = "max-resources-per-nuke"
)

// MCPCommand returns the mcp-server CLI command definition.
func MCPCommand(version string) *cli.Command {
	return &cli.Command{
		Name:  "mcp-server",
		Usage: "Start an MCP server over stdio for AI agent integration",
		Action: errors.WithPanicHandling(func(c *cli.Context) error {
			// Apply log level from flag/env
			if logLevel := c.String(FlagLogLevel); logLevel != "" {
				if err := logging.ParseLogLevel(logLevel); err != nil {
					return errors.WithStackTrace(err)
				}
			}

			cfg := mcp.DefaultServerConfig()
			cfg.ReadOnly = c.Bool(FlagMCPReadOnly)
			cfg.AllowedRegions = c.StringSlice(FlagMCPAllowedRegions)
			cfg.AllowedResourceTypes = c.StringSlice(FlagMCPAllowedResourceTypes)
			cfg.AllowedProjects = c.StringSlice(FlagMCPAllowedProjects)

			if c.IsSet(FlagMCPMaxResourcesPerNuke) {
				v := c.Int(FlagMCPMaxResourcesPerNuke)
				if v < 1 {
					v = 1
				}
				cfg.MaxResourcesPerNuke = v
			}

			s := mcp.NewServer(version, cfg)
			return s.Serve()
		}),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  FlagMCPReadOnly,
				Usage: "Disable nuke operations (only allow inspect and list)",
			},
			&cli.StringSliceFlag{
				Name:  FlagMCPAllowedRegions,
				Usage: "Whitelist of allowed AWS regions. If empty, all regions are allowed.",
			},
			&cli.StringSliceFlag{
				Name:  FlagMCPAllowedResourceTypes,
				Usage: "Whitelist of allowed resource types. If empty, all types are allowed.",
			},
			&cli.StringSliceFlag{
				Name:  FlagMCPAllowedProjects,
				Usage: "Whitelist of allowed GCP project IDs. If empty, all projects are allowed.",
			},
			&cli.IntFlag{
				Name:  FlagMCPMaxResourcesPerNuke,
				Value: mcp.DefaultMaxResourcesPerNuke,
				Usage: "Maximum number of resources that can be nuked in a single operation (minimum: 1)",
			},
			&cli.StringFlag{
				Name:    FlagLogLevel,
				Value:   "warn",
				Usage:   "Set log level (logs go to stderr in MCP mode)",
				EnvVars: []string{"LOG_LEVEL"},
			},
		},
	}
}
