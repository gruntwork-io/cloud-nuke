package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// CreateCli - Create the CLI app with all commands, flags, and usage text configured.
func CreateCli(version string, mixPanelClientId string) *cli.App {
	app := cli.NewApp()
	logging.InitLogger("cloud-nuke", version)
	_, disableTelemetryFlag := os.LookupEnv("DISABLE_TELEMETRY")
	if !disableTelemetryFlag {
		logging.Logger.Error("This program sends telemetry to Gruntwork. To disable, set DISABLE_TELEMETRY=true as an environment variable")
	}
	telemetry.InitTelemetry("cloud-nuke", version, mixPanelClientId)
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
			},
		},
	}

	return app
}

func parseDurationParam(paramValue string) (*time.Time, error) {
	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// make it negative so it goes back in time
	duration = -1 * duration

	excludeAfter := time.Now().Add(duration)
	return &excludeAfter, nil
}

func parseLogLevel(c *cli.Context) error {
	logLevel := c.String("log-level")

	parsedLogLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Invalid log level - %s - %s", logLevel, err)
	}
	logging.Logger.Level = parsedLogLevel
	return nil
}

func awsNuke(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws",
	}, map[string]interface{}{})

	parseErr := parseLogLevel(c)
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

	if c.Bool("list-resource-types") {
		for _, resourceType := range aws.ListResourceTypes() {
			fmt.Println(resourceType)
		}
		return nil
	}

	// Ensure that the resourceTypes and excludeResourceTypes arguments are valid, and then filter
	// resourceTypes
	resourceTypes, err := aws.HandleResourceTypeSelections(c.StringSlice("resource-type"), c.StringSlice("exclude-resource-type"))
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Bad Resource Types",
		}, map[string]interface{}{})
		return err
	}

	targetedResourceList := []pterm.BulletListItem{}

	for _, resource := range resourceTypes {
		targetedResourceList = append(targetedResourceList, pterm.BulletListItem{Level: 0, Text: resource})
	}

	ui.WarningMessage("The following resource types are targeted for destruction")

	// Log which resource types will be nuked
	list := pterm.DefaultBulletList.
		WithItems(targetedResourceList).
		WithBullet(ui.TargetEmoji)

	renderErr := list.Render()
	if renderErr != nil {
		return errors.WithStackTrace(renderErr)
	}

	regions, err := aws.GetEnabledRegions()
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting regions",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	// global is a fake region, used to represent global resources
	regions = append(regions, aws.GlobalRegion)

	selectedRegions := c.StringSlice("region")
	excludedRegions := c.StringSlice("exclude-region")

	// targetRegions uses selectedRegions and excludedRegions to create a final
	// target region slice.
	targetRegions, err := aws.GetTargetRegions(regions, selectedRegions, excludedRegions)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error targeting regions",
		}, map[string]interface{}{})
		return fmt.Errorf("Failed to select regions: %s", err)
	}

	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error parsing duration",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	spinnerMsg := fmt.Sprintf("Retrieving active AWS resources in [%s]", strings.Join(targetRegions[:], ", "))

	// Start a simple spinner to track progress reading all relevant AWS resources
	spinnerSuccess, spinnerErr := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(spinnerMsg)

	if spinnerErr != nil {
		return errors.WithStackTrace(spinnerErr)
	}

	account, err := aws.GetAllResources(targetRegions, *excludeAfter, resourceTypes, configObj, c.Bool("delete-unaliased-kms-keys"))
	// Stop the spinner
	spinnerSuccess.Stop()
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

	nukableResources := aws.ExtractResourcesForPrinting(account)

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Found resources to nuke",
	}, map[string]interface{}{
		"totalResourceCount": len(nukableResources),
	})

	ui.WarningMessage(fmt.Sprintf("The following %d AWS resources will be nuked:\n", len(nukableResources)))

	items := []pterm.BulletListItem{}

	for _, resource := range nukableResources {
		items = append(items, pterm.BulletListItem{Level: 0, Text: resource})
	}

	targetList := pterm.DefaultBulletList.
		WithItems(items).
		WithBullet(ui.FireEmoji)

	targetRenderErr := targetList.Render()
	if targetRenderErr != nil {
		return errors.WithStackTrace(targetRenderErr)
	}

	if c.Bool("dry-run") {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Skipping nuke, dryrun set",
		}, map[string]interface{}{})
		logging.Logger.Infoln("Not taking any action as dry-run set to true.")
		return nil
	}

	if !c.Bool("force") {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Awaiting nuke confirmation",
		}, map[string]interface{}{})
		prompt := "\nAre you sure you want to nuke all listed resources? Enter 'nuke' to confirm (or exit with ^C) "
		proceed, err := confirmationPrompt(prompt, 2)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error confirming nuke",
			}, map[string]interface{}{})
			return err
		}
		if proceed {
			if err := aws.NukeAllResources(account, targetRegions); err != nil {
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
		logging.Logger.Infoln("The --force flag is set, so waiting for 10 seconds before proceeding to nuke everything in your account. If you don't want to proceed, hit CTRL+C now!!")
		for i := 10; i > 0; i-- {
			fmt.Printf("%d...", i)
			time.Sleep(1 * time.Second)
		}

		if err := aws.NukeAllResources(account, targetRegions); err != nil {
			return err
		}
	}

	ui.RenderRunReport()

	return nil
}

func awsDefaults(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws-defaults",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws-defaults",
	}, map[string]interface{}{})
	parseErr := parseLogLevel(c)
	if parseErr != nil {
		return errors.WithStackTrace(parseErr)
	}

	logging.Logger.Infoln("Identifying enabled regions")
	regions, err := aws.GetEnabledRegions()
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, region := range regions {
		logging.Logger.Infof("Found enabled region %s", region)
	}

	selectedRegions := c.StringSlice("region")
	excludedRegions := c.StringSlice("exclude-region")

	// targetRegions uses selectedRegions and excludedRegions to create a final
	// target region slice.
	targetRegions, err := aws.GetTargetRegions(regions, selectedRegions, excludedRegions)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting target regions",
		}, map[string]interface{}{})
		return fmt.Errorf("Failed to select regions: %s", err)
	}

	if c.Bool("sg-only") {
		logging.Logger.Info("Not removing default VPCs.")
	} else {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Nuking default VPCs",
		}, map[string]interface{}{})
		err = nukeDefaultVpcs(c, targetRegions)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error nuking default vpcs",
			}, map[string]interface{}{})
			return errors.WithStackTrace(err)
		}
	}
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Nuking default security groups",
	}, map[string]interface{}{})
	err = nukeDefaultSecurityGroups(c, targetRegions)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error nuking default security groups",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}
	ui.RenderRunReport()

	return nil
}

func nukeDefaultVpcs(c *cli.Context, regions []string) error {
	// Start a spinner so the user knows we're still performing work in the background
	spinnerMsg := "Discovering default VPCs"

	spinnerSuccess, spinnerErr := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(spinnerMsg)

	if spinnerErr != nil {
		return errors.WithStackTrace(spinnerErr)
	}

	vpcPerRegion := aws.NewVpcPerRegion(regions)
	vpcPerRegion, err := aws.GetDefaultVpcs(vpcPerRegion)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Stop the spinner
	spinnerSuccess.Stop()

	if len(vpcPerRegion) == 0 {
		logging.Logger.Info("No default VPCs found.")
		return nil
	}

	targetedRegionList := []pterm.BulletListItem{}

	for _, vpc := range vpcPerRegion {
		vpcDetailString := fmt.Sprintf("Default VPC %s %s", vpc.VpcId, vpc.Region)
		targetedRegionList = append(targetedRegionList, pterm.BulletListItem{Level: 0, Text: vpcDetailString})
	}

	ui.WarningMessage("The following Default VPCs are targeted for destruction")

	// Log which Default VPCs will be nuked
	list := pterm.DefaultBulletList.
		WithItems(targetedRegionList).
		WithBullet(ui.TargetEmoji)

	renderErr := list.Render()
	if renderErr != nil {
		return errors.WithStackTrace(renderErr)
	}

	var proceed bool
	if !c.Bool("force") {
		proceed, err = confirmationPrompt("Are you sure you want to nuke the default VPCs listed above? Enter 'nuke' to confirm (or exit with ^C)", 2)
		if err != nil {
			return err
		}
	}

	if proceed || c.Bool("force") {
		// Start nuke progress bar with correct number of items
		aws.StartProgressBarWithLength(len(targetedRegionList))
		err := aws.NukeVpcs(vpcPerRegion)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	return nil
}

func nukeDefaultSecurityGroups(c *cli.Context, regions []string) error {
	// Start a spinner so the user knows we're still performing work in the background
	spinnerMsg := "Discovering default security groups"

	spinnerSuccess, spinnerErr := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(spinnerMsg)

	if spinnerErr != nil {
		return errors.WithStackTrace(spinnerErr)
	}

	defaultSgs, err := aws.GetDefaultSecurityGroups(regions)
	spinnerSuccess.Stop()
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(defaultSgs) == 0 {
		logging.Logger.Info("No default security groups found.")
		return nil
	}

	targetedRegionList := []pterm.BulletListItem{}

	for _, sg := range defaultSgs {
		defaultSgDetailText := fmt.Sprintf("* Default rules for SG %s %s %s", sg.GroupId, sg.GroupName, sg.Region)
		targetedRegionList = append(targetedRegionList, pterm.BulletListItem{Level: 0, Text: defaultSgDetailText})
	}

	ui.WarningMessage("The following Default security group rules are targeted for destruction")

	// Log which default security group rules will be nuked
	list := pterm.DefaultBulletList.
		WithItems(targetedRegionList).
		WithBullet(ui.TargetEmoji)

	renderErr := list.Render()
	if renderErr != nil {
		return errors.WithStackTrace(renderErr)
	}

	var proceed bool
	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke the rules in these default security groups ? Enter 'nuke' to confirm (or exit with ^C)"
		proceed, err = confirmationPrompt(prompt, 2)
		if err != nil {
			return err
		}
	}

	if proceed || c.Bool("force") {
		err := aws.NukeDefaultSecurityGroupRules(defaultSgs)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}

	return nil
}

func confirmationPrompt(prompt string, maxPrompts int) (bool, error) {
	prompts := 0

	ui.UrgentMessage("THE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

	for prompts < maxPrompts {
		confirmPrompt := pterm.DefaultInteractiveTextInput.WithMultiLine(false)
		pterm.Println()
		input, err := confirmPrompt.Show(prompt)
		if err != nil {
			logging.Logger.Errorf("[Failed to render prompt] %s", err)
			return false, errors.WithStackTrace(err)
		}
		response := strings.ToLower(strings.TrimSpace(input))
		if response == "nuke" {
			return true, nil
		}
		fmt.Printf("Invalid value '%s' was entered.\n", input)
		prompts++
		pterm.Println()
	}

	return false, nil
}

func awsInspect(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws-inspect",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws-inspect",
	}, map[string]interface{}{})
	logging.Logger.Infoln("Identifying enabled regions")
	regions, err := aws.GetEnabledRegions()
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, region := range regions {
		logging.Logger.Infof("Found enabled region %s", region)
	}

	if c.Bool("list-resource-types") {
		for _, resourceType := range aws.ListResourceTypes() {
			logging.Logger.Infoln(resourceType)
		}
		return nil
	}

	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error parsing duration",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	query, err := aws.NewQuery(
		c.StringSlice("region"),
		c.StringSlice("exclude-region"),
		c.StringSlice("resource-type"),
		c.StringSlice("exclude-resource-type"),
		*excludeAfter,
		c.Bool("list-unaliased-kms-keys"),
	)
	if err != nil {
		return aws.QueryCreationError{Underlying: err}
	}

	accountResources, err := aws.InspectResources(query)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
	}

	foundResources := aws.ExtractResourcesForPrinting(accountResources)

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Found resources with aws-inspect",
	}, map[string]interface{}{
		"resourceCount": len(foundResources),
	})
	for _, resource := range foundResources {
		logging.Logger.Infoln(resource)
	}

	return nil
}
