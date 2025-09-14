package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

// CreateCli - Create the CLI app with all commands, flags, and usage text configured.
func CreateCli(version string) *cli.App {
	app := cli.NewApp()
	_, disableTelemetryFlag := os.LookupEnv("DISABLE_TELEMETRY")
	if !disableTelemetryFlag {
		pterm.Warning.Println("This program sends telemetry to Gruntwork. To disable, set DISABLE_TELEMETRY=true as an environment variable")
		pterm.Println()
	}
	telemetry.InitTelemetry("cloud-nuke", version)
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "initialized",
	}, map[string]interface{}{})
	app.Name = "cloud-nuke"
	app.HelpName = app.Name
	app.Authors = []*cli.Author{
		{Name: "Gruntwork", Email: "www.gruntwork.io"},
	}
	app.Version = version
	app.Usage = "A CLI tool to nuke (delete) cloud resources."
	app.Commands = []*cli.Command{
		{
			Name:   "aws",
			Usage:  "BEWARE: DESTRUCTIVE OPERATION! Nukes AWS resources (ASG, ELB, ELBv2, EBS, EC2, AMI, Snapshots, Elastic IP, RDS, Lambda Function).",
			Action: errors.WithPanicHandling(awsNuke),
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "region",
					Usage: "Regions to include. Include multiple times if more than one.",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "Regions to exclude. Include multiple times if more than one.",
				},
				&cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to nuke. Include multiple times if more than one.",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from nuking. Include multiple times if more than one.",
				},
				&cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				&cli.StringFlag{
					Name:  "older-than",
					Usage: "Only delete resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.StringFlag{
					Name:  "newer-than",
					Usage: "Only delete resources newer than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Dry run without taking any action.",
				},
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Skip nuke confirmation prompt. WARNING: this will automatically delete all targeted resources without any confirmation. It will not modify resource selections made via the --resource-type flag or an optional config file.",
				},
				&cli.StringFlag{
					Name:    "log-level",
					Value:   "info",
					Usage:   "Set log level",
					EnvVars: []string{"LOG_LEVEL"},
				},
				&cli.BoolFlag{
					Name:  "delete-unaliased-kms-keys",
					Usage: "Delete KMS keys that do not have aliases associated with them.",
				},
				&cli.StringFlag{
					Name:  "config",
					Usage: "YAML file specifying matching rules.",
				},
				&cli.StringFlag{
					Name:  "timeout",
					Usage: "Resource execution timeout.",
				},
				&cli.BoolFlag{
					Name:  "exclude-first-seen",
					Usage: "Set a flag for excluding first-seen-tag",
				},
				&cli.StringFlag{
					Name:  "output-format",
					Usage: "Output format (table, json)",
					Value: "table",
				},
				&cli.StringFlag{
					Name:  "output-file",
					Usage: "Write output to file instead of stdout (optional)",
				},
			},
		}, {
			Name:   "gcp",
			Usage:  "BEWARE: DESTRUCTIVE OPERATION! Nukes GCP resources (GCS Buckets, Compute Instances, Cloud SQL, etc.).",
			Action: errors.WithPanicHandling(gcpNuke),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "project-id",
					Usage:    "GCP Project ID to nuke resources from.",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to nuke. Include multiple times if more than one.",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from nuking. Include multiple times if more than one.",
				},
				&cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				&cli.StringFlag{
					Name:  "older-than",
					Usage: "Only delete resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.StringFlag{
					Name:  "newer-than",
					Usage: "Only delete resources newer than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Dry run without taking any action.",
				},
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Skip nuke confirmation prompt. WARNING: this will automatically delete all targeted resources without any confirmation. It will not modify resource selections made via the --resource-type flag or an optional config file.",
				},
				&cli.StringFlag{
					Name:    "log-level",
					Value:   "info",
					Usage:   "Set log level",
					EnvVars: []string{"LOG_LEVEL"},
				},
				&cli.StringFlag{
					Name:  "config",
					Usage: "YAML file specifying matching rules.",
				},
				&cli.StringFlag{
					Name:  "timeout",
					Usage: "Resource execution timeout.",
				},
				&cli.StringFlag{
					Name:  "output-format",
					Usage: "Output format (table, json)",
					Value: "table",
				},
				&cli.StringFlag{
					Name:  "output-file",
					Usage: "Write output to file instead of stdout (optional)",
				},
			},
		}, {
			Name:   "inspect-gcp",
			Usage:  "Non-destructive inspection of target GCP resources only",
			Action: errors.WithPanicHandling(gcpInspect),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "project-id",
					Usage:    "GCP Project ID to inspect resources from.",
					Required: true,
				},
				&cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				&cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to inspect. Include multiple times if more than one.",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from inspection. Include multiple times if more than one.",
				},
				&cli.StringFlag{
					Name:  "older-than",
					Usage: "Only inspect resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.StringFlag{
					Name:  "newer-than",
					Usage: "Only inspect resources newer than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.StringFlag{
					Name:    "log-level",
					Value:   "info",
					Usage:   "Set log level",
					EnvVars: []string{"LOG_LEVEL"},
				},
				&cli.StringFlag{
					Name:  "output-format",
					Usage: "Output format (table, json)",
					Value: "table",
				},
				&cli.StringFlag{
					Name:  "output-file",
					Usage: "Write output to file instead of stdout (optional)",
				},
			},
		}, {
			Name:   "defaults-aws",
			Usage:  "Nukes AWS default VPCs and permissive default security group rules. Optionally include/exclude specified regions, or just nuke security group rules (not default VPCs).",
			Action: errors.WithPanicHandling(awsDefaults),
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "region",
					Usage: "regions to include",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "regions to exclude",
				},
				&cli.BoolFlag{
					Name:  "sg-only",
					Usage: "Destroy default security group rules only. Do not destroy default VPCs.",
				},
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Skip confirmation prompt. WARNING: this will automatically delete defaults without any confirmation",
				},
				&cli.StringFlag{
					Name:    "log-level",
					Value:   "info",
					Usage:   "Set log level",
					EnvVars: []string{"LOG_LEVEL"},
				},
			},
		}, {
			Name:   "inspect-aws",
			Usage:  "Non-destructive inspection of target resources only",
			Action: errors.WithPanicHandling(awsInspect),
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "region",
					Usage: "regions to include",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "regions to exclude",
				},
				&cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				&cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to inspect. Include multiple times if more than one.",
				},
				&cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from inspection. Include multiple times if more than one.",
				},
				&cli.StringFlag{
					Name:  "older-than",
					Usage: "Only inspect resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.StringFlag{
					Name:  "newer-than",
					Usage: "Only delete resources newer than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				&cli.BoolFlag{
					Name:  "list-unaliased-kms-keys",
					Usage: "List KMS keys that do not have aliases associated with them.",
				},
				&cli.StringFlag{
					Name:    "log-level",
					Value:   "info",
					Usage:   "Set log level",
					EnvVars: []string{"LOG_LEVEL"},
				},
				&cli.BoolFlag{
					Name:  "exclude-first-seen",
					Usage: "Set a flag for excluding first-seen-tag",
				},
				&cli.StringFlag{
					Name:  "output-format",
					Usage: "Output format (table, json)",
					Value: "table",
				},
				&cli.StringFlag{
					Name:  "output-file",
					Usage: "Write output to file instead of stdout (optional)",
				},
			},
		},
	}

	return app
}

func parseDurationParam(paramValue string) (*time.Time, error) {
	if paramValue == "0s" || paramValue == "" {
		return nil, nil
	}

	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// make it negative so it goes back in time
	duration = -1 * duration

	excludeAfter := time.Now().Add(duration)
	return &excludeAfter, nil
}

func parseTimeoutDurationParam(paramValue string) (*time.Duration, error) {
	if paramValue == "0s" || paramValue == "" {
		return nil, nil
	}

	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return &duration, nil
}

func awsNuke(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws",
	}, map[string]interface{}{})

	// Handle the case where the user only wants to list resource types
	if c.Bool("list-resource-types") {
		return handleListResourceTypes()
	}

	parseErr := logging.ParseLogLevel(c.String("log-level"))
	if parseErr != nil {
		return errors.WithStackTrace(parseErr)
	}

	configObj := config.Config{}
	configFilePath := c.String("config")
	if configFilePath != "" {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Reading config file",
		}, map[string]interface{}{})
		configObjPtr, err := config.GetConfig(configFilePath)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error reading config file",
			}, map[string]interface{}{})
			return fmt.Errorf("Error reading config - %s - %s", configFilePath, err)
		}
		configObj = *configObjPtr
	}

	query, err := generateQuery(c, c.Bool("delete-unaliased-kms-keys"), nil, false)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return awsNukeHelper(c, configObj, query)
}

func awsDefaults(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws-defaults",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws-defaults",
	}, map[string]interface{}{})
	parseErr := logging.ParseLogLevel(c.String("log-level"))
	if parseErr != nil {
		return errors.WithStackTrace(parseErr)
	}

	// Depending on the flag, we need to populate resource-types to include/exclude
	resourceTypes := []string{"vpc"} // nuking the vpc will remove the attached default security groups
	if c.Bool("sg-only") {
		resourceTypes = []string{"security-group"}
	}

	query, err := generateQuery(c, false, resourceTypes, true)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Note: config feature only available for awsNuke command.
	return awsNukeHelper(c, config.Config{}, query)
}

func awsInspect(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws-inspect",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws-inspect",
	}, map[string]interface{}{})

	// Handle the case where the user only wants to list resource types
	if c.Bool("list-resource-types") {
		return handleListResourceTypes()
	}

	query, err := generateQuery(c, c.Bool("list-unaliased-kms-keys"), nil, false)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	outputFormat := c.String("output-format")
	outputFile := c.String("output-file")

	_, err = handleGetResourcesWithFormat(c, config.Config{}, query, outputFormat, outputFile)
	return err
}

func awsNukeHelper(c *cli.Context, configObj config.Config, query *aws.Query) error {
	account, err := handleGetResources(c, configObj, query)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting resources",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	if len(account.Resources) == 0 {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "No resources to nuke",
		}, map[string]interface{}{})
		pterm.Info.Println("Nothing to nuke, you're all good!")
		return nil
	}

	if c.Bool("dry-run") {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Skipping nuke, dryrun set",
		}, map[string]interface{}{})
		logging.Info("Not taking any action as dry-run set to true.")
		return nil
	}

	if !c.Bool("force") {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Awaiting nuke confirmation",
		}, map[string]interface{}{})
		prompt := "\nAre you sure you want to nuke all listed resources? Enter 'nuke' to confirm (or exit with ^C) "
		proceed, err := ui.RenderNukeConfirmationPrompt(prompt, 2)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error confirming nuke",
			}, map[string]interface{}{})
			return err
		}
		if proceed {
			if err := aws.NukeAllResources(account, query.Regions); err != nil {
				return err
			}
		} else {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "User aborted nuke",
			}, map[string]interface{}{})
		}
	} else {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Forcing nuke in 10 seconds",
		}, map[string]interface{}{})
		logging.Info("The --force flag is set, so waiting for 10 seconds before proceeding to nuke everything in your account. If you don't want to proceed, hit CTRL+C now!!")
		for i := 10; i > 0; i-- {
			fmt.Printf("%d...", i)
			time.Sleep(1 * time.Second)
		}

		if err := aws.NukeAllResources(account, query.Regions); err != nil {
			return err
		}
	}

	outputFormat := c.String("output-format")
	outputFile := c.String("output-file")
	ui.RenderRunReportWithFormat(outputFormat, outputFile)
	return nil
}

func generateQuery(c *cli.Context, includeUnaliasedKmsKeys bool, overridingResourceTypes []string, onlyDefault bool) (*aws.Query, error) {
	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	includeAfter, err := parseDurationParam(c.String("newer-than"))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	timeout, err := parseTimeoutDurationParam(c.String("timeout"))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	resourceTypes := c.StringSlice("resource-type")
	if overridingResourceTypes != nil {
		resourceTypes = overridingResourceTypes
	}

	return aws.NewQuery(
		c.StringSlice("region"),
		c.StringSlice("exclude-region"),
		resourceTypes,
		c.StringSlice("exclude-resource-type"),
		excludeAfter,
		includeAfter,
		includeUnaliasedKmsKeys,
		timeout,
		onlyDefault,
		c.Bool("exclude-first-seen"),
	)
}

func handleGetResources(c *cli.Context, configObj config.Config, query *aws.Query) (
	*aws.AwsAccountResources, error) {
	return handleGetResourcesWithFormat(c, configObj, query, "table", "")
}

func handleGetResourcesWithFormat(c *cli.Context, configObj config.Config, query *aws.Query, outputFormat string, outputFile string) (
	*aws.AwsAccountResources, error) {

	// Only show progress output for table format
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("AWS Resource Query Parameters")
		err := ui.RenderQueryAsBulletList(query)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		pterm.Println()
	}

	accountResources, err := aws.GetAllResources(c.Context, query, configObj)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return nil, errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
	}

	// Only show section header for table format
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Found AWS Resources")
	}

	err = ui.RenderResourcesAsTableWithFormat(accountResources, query, outputFormat, outputFile)

	return accountResources, err
}

func handleListResourceTypes() error {
	// Handle the case where the user only wants to list resource types
	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("AWS Resource Types")
	return ui.RenderResourceTypesAsBulletList(aws.ListResourceTypes())
}

func gcpNuke(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start gcp",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End gcp",
	}, map[string]interface{}{})

	// Handle the case where the user only wants to list resource types
	if c.Bool("list-resource-types") {
		return handleListGcpResourceTypes()
	}

	parseErr := logging.ParseLogLevel(c.String("log-level"))
	if parseErr != nil {
		return errors.WithStackTrace(parseErr)
	}

	configObj := config.Config{}
	configFilePath := c.String("config")
	if configFilePath != "" {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Reading config file",
		}, map[string]interface{}{})
		configObjPtr, err := config.GetConfig(configFilePath)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error reading config file",
			}, map[string]interface{}{})
			return fmt.Errorf("Error reading config - %s - %s", configFilePath, err)
		}
		configObj = *configObjPtr
	}

	projectID := c.String("project-id")
	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	includeAfter, err := parseDurationParam(c.String("newer-than"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	timeout, err := parseTimeoutDurationParam(c.String("timeout"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Apply timeout to config if specified
	if timeout != nil {
		configObj.AddTimeout(timeout)
	}

	// Apply time filters to config
	if excludeAfter != nil {
		configObj.AddExcludeAfterTime(excludeAfter)
	}
	if includeAfter != nil {
		configObj.AddIncludeAfterTime(includeAfter)
	}

	return gcpNukeHelper(c, configObj, projectID)
}

func gcpInspect(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start gcp-inspect",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End gcp-inspect",
	}, map[string]interface{}{})

	// Handle the case where the user only wants to list resource types
	if c.Bool("list-resource-types") {
		return handleListGcpResourceTypes()
	}

	parseErr := logging.ParseLogLevel(c.String("log-level"))
	if parseErr != nil {
		return errors.WithStackTrace(parseErr)
	}

	projectID := c.String("project-id")
	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	includeAfter, err := parseDurationParam(c.String("newer-than"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	configObj := config.Config{}

	// Apply time filters to config
	if excludeAfter != nil {
		configObj.AddExcludeAfterTime(excludeAfter)
	}
	if includeAfter != nil {
		configObj.AddIncludeAfterTime(includeAfter)
	}

	outputFormat := c.String("output-format")
	outputFile := c.String("output-file")

	_, err = handleGetGcpResourcesWithFormat(c, configObj, projectID, outputFormat, outputFile)
	return err
}

func gcpNukeHelper(c *cli.Context, configObj config.Config, projectID string) error {
	account, err := handleGetGcpResources(c, configObj, projectID)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting resources",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	if len(account.Resources) == 0 {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "No resources to nuke",
		}, map[string]interface{}{})
		pterm.Info.Println("Nothing to nuke, you're all good!")
		return nil
	}

	if c.Bool("dry-run") {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Skipping nuke, dryrun set",
		}, map[string]interface{}{})
		logging.Info("Not taking any action as dry-run set to true.")
		return nil
	}

	if !c.Bool("force") {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Awaiting nuke confirmation",
		}, map[string]interface{}{})
		prompt := "\nAre you sure you want to nuke all listed GCP resources? Enter 'nuke' to confirm (or exit with ^C) "
		proceed, err := ui.RenderNukeConfirmationPrompt(prompt, 2)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error confirming nuke",
			}, map[string]interface{}{})
			return err
		}
		if proceed {
			gcp.NukeAllResources(account, nil)
		} else {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "User aborted nuke",
			}, map[string]interface{}{})
		}
	} else {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Forcing nuke in 10 seconds",
		}, map[string]interface{}{})
		logging.Info("The --force flag is set, so waiting for 10 seconds before proceeding to nuke everything in your GCP project. If you don't want to proceed, hit CTRL+C now!!")
		for i := 10; i > 0; i-- {
			fmt.Printf("%d...", i)
			time.Sleep(1 * time.Second)
		}

		gcp.NukeAllResources(account, nil)
	}

	outputFormat := c.String("output-format")
	outputFile := c.String("output-file")
	ui.RenderRunReportWithFormat(outputFormat, outputFile)
	return nil
}

func handleGetGcpResources(c *cli.Context, configObj config.Config, projectID string) (
	*gcp.GcpProjectResources, error) {
	return handleGetGcpResourcesWithFormat(c, configObj, projectID, "table", "")
}

func handleGetGcpResourcesWithFormat(c *cli.Context, configObj config.Config, projectID string, outputFormat string, outputFile string) (
	*gcp.GcpProjectResources, error) {

	// Only show progress output for table format
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("GCP Resource Query Parameters")
		pterm.Println("Project ID:", projectID)
		pterm.Println()
	}

	accountResources, err := gcp.GetAllResources(projectID, configObj, time.Time{}, time.Time{})
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return nil, errors.WithStackTrace(err)
	}

	// Only show section header for table format
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Found GCP Resources")
	}

	err = ui.RenderGcpResourcesAsTableWithFormat(accountResources, outputFormat, outputFile)

	return accountResources, err
}

func handleListGcpResourceTypes() error {
	// Handle the case where the user only wants to list resource types
	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("GCP Resource Types")
	return ui.RenderResourceTypesAsBulletList(gcp.ListResourceTypes())
}
