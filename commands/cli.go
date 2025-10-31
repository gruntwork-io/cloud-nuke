package commands

import (
	"os"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

// CreateCli creates and configures the CLI application with all commands, flags, and usage text.
// This is the main entry point for setting up the cloud-nuke CLI interface.
func CreateCli(version string) *cli.App {
	app := cli.NewApp()

	// Display telemetry warning unless explicitly disabled
	_, disableTelemetryFlag := os.LookupEnv("DISABLE_TELEMETRY")
	if !disableTelemetryFlag {
		pterm.Warning.Println("This program sends telemetry to Gruntwork. To disable, set DISABLE_TELEMETRY=true as an environment variable")
		pterm.Println()
	}

	// Initialize telemetry for usage tracking
	telemetry.InitTelemetry("cloud-nuke", version)
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: telemetry.EventInitialized,
	}, map[string]interface{}{})

	// Configure basic app metadata
	app.Name = "cloud-nuke"
	app.HelpName = app.Name
	app.Authors = []*cli.Author{
		{Name: "Gruntwork", Email: "www.gruntwork.io"},
	}
	app.Version = version
	app.Usage = "A CLI tool to nuke (delete) cloud resources."

	// Register all available commands
	app.Commands = []*cli.Command{
		{
			Name:   "aws",
			Usage:  "BEWARE: DESTRUCTIVE OPERATION! Nukes AWS resources.",
			Action: errors.WithPanicHandling(awsNuke),
			Flags: CombineFlags(
				RegionFlags(),
				CommonResourceTypeFlags(),
				CommonTimeFlags(),
				CommonExecutionFlags(),
				CommonOutputFlags(),
				[]cli.Flag{
					ConfigFlag(),
					&cli.BoolFlag{
						Name:  FlagDeleteUnaliasedKMSKeys,
						Usage: "Delete KMS keys that do not have aliases associated with them.",
					},
					&cli.BoolFlag{
						Name:  FlagExcludeFirstSeen,
						Usage: "Set a flag for excluding first-seen-tag",
					},
				},
			),
		}, {
			Name:   "gcp",
			Usage:  "BEWARE: DESTRUCTIVE OPERATION! Nukes GCP resources.",
			Action: errors.WithPanicHandling(gcpNuke),
			Flags: CombineFlags(
				[]cli.Flag{GCPProjectFlag()},
				CommonResourceTypeFlags(),
				CommonTimeFlags(),
				CommonExecutionFlags(),
				CommonOutputFlags(),
				[]cli.Flag{ConfigFlag()},
			),
		}, {
			Name:   "inspect-gcp",
			Usage:  "Non-destructive inspection of target GCP resources only",
			Action: errors.WithPanicHandling(gcpInspect),
			Flags: CombineFlags(
				[]cli.Flag{GCPProjectFlag()},
				InspectResourceTypeFlags(),
				CommonTimeFlags(),
				CommonOutputFlags(),
			),
		}, {
			Name:   "defaults-aws",
			Usage:  "Nukes AWS default VPCs and permissive default security group rules.",
			Action: errors.WithPanicHandling(awsDefaults),
			Flags: CombineFlags(
				RegionFlags(),
				[]cli.Flag{
					&cli.BoolFlag{
						Name:  FlagSGOnly,
						Usage: "Destroy default security group rules only. Do not destroy default VPCs.",
					},
					&cli.BoolFlag{
						Name:  FlagForce,
						Usage: "Skip confirmation prompt. WARNING: this will automatically delete defaults without any confirmation",
					},
					&cli.StringFlag{
						Name:    FlagLogLevel,
						Value:   DefaultLogLevel,
						Usage:   "Set log level",
						EnvVars: []string{"LOG_LEVEL"},
					},
				},
			),
		}, {
			Name:   "inspect-aws",
			Usage:  "Non-destructive inspection of target resources only",
			Action: errors.WithPanicHandling(awsInspect),
			Flags: CombineFlags(
				RegionFlags(),
				InspectResourceTypeFlags(),
				CommonTimeFlags(),
				CommonOutputFlags(),
				[]cli.Flag{
					&cli.BoolFlag{
						Name:  FlagListUnaliasedKMSKeys,
						Usage: "List KMS keys that do not have aliases associated with them.",
					},
					&cli.BoolFlag{
						Name:  FlagExcludeFirstSeen,
						Usage: "Set a flag for excluding first-seen-tag",
					},
				},
			),
		},
	}

	return app
}
