package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/progressbar"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/shell"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// CreateCli - Create the CLI app with all commands, flags, and usage text configured.
func CreateCli(version string) *cli.App {
	app := cli.NewApp()

	app.Name = "cloud-nuke"
	app.HelpName = app.Name
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.Version = version
	app.Usage = "A CLI tool to nuke (delete) cloud resources."
	app.Commands = []cli.Command{
		{
			Name:   "aws",
			Usage:  "BEWARE: DESTRUCTIVE OPERATION! Nukes AWS resources (ASG, ELB, ELBv2, EBS, EC2, AMI, Snapshots, Elastic IP, RDS, Lambda Function).",
			Action: errors.WithPanicHandling(awsNuke),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "region",
					Usage: "Regions to include. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "Regions to exclude. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to nuke. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from nuking. Include multiple times if more than one.",
				},
				cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				cli.StringFlag{
					Name:  "older-than",
					Usage: "Only delete resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Dry run without taking any action.",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Skip nuke confirmation prompt. WARNING: this will automatically delete all targeted resources without any confirmation. It will not modify resource selections made via the --resource-type flag or an optional config file.",
				},
				cli.StringFlag{
					Name:   "log-level",
					Value:  "info",
					Usage:  "Set log level",
					EnvVar: "LOG_LEVEL",
				},
				cli.StringFlag{
					Name:  "config",
					Usage: "YAML file specifying matching rules.",
				},
			},
		}, {
			Name:   "defaults-aws",
			Usage:  "Nukes AWS default VPCs and permissive default security group rules. Optionally include/exclude specified regions, or just nuke security group rules (not default VPCs).",
			Action: errors.WithPanicHandling(awsDefaults),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "region",
					Usage: "regions to include",
				},
				cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "regions to exclude",
				},
				cli.BoolFlag{
					Name:  "sg-only",
					Usage: "Destroy default security group rules only. Do not destroy default VPCs.",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Skip confirmation prompt. WARNING: this will automatically delete defaults without any confirmation",
				},
			},
		}, {
			Name:   "inspect-aws",
			Usage:  "Non-destructive inspection of target resources only",
			Action: errors.WithPanicHandling(awsInspect),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "region",
					Usage: "regions to include",
				},
				cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "regions to exclude",
				},
				cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to inspect. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from inspection. Include multiple times if more than one.",
				},
				cli.StringFlag{
					Name:  "older-than",
					Usage: "Only inspect resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
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

func awsNuke(c *cli.Context) error {
	logLevel := c.String("log-level")

	parsedLogLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Invalid log level - %s - %s", logLevel, err)
	}
	logging.Logger.Level = parsedLogLevel

	configObj := config.Config{}
	configFilePath := c.String("config")

	if configFilePath != "" {
		configObjPtr, err := config.GetConfig(configFilePath)
		if err != nil {
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
		return fmt.Errorf("Failed to select regions: %s", err)
	}

	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
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

	account, err := aws.GetAllResources(targetRegions, *excludeAfter, resourceTypes, configObj)
	// Stop the spinner
	spinnerSuccess.Stop()
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(account.Resources) == 0 {
		pterm.Info.Println("Nothing to nuke, you're all good!")
		return nil
	}

	nukableResources := aws.ExtractResourcesForPrinting(account)

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
		logging.Logger.Infoln("Not taking any action as dry-run set to true.")
		return nil
	}

	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke all listed resources? Enter 'nuke' to confirm (or exit with ^C): "
		proceed, err := confirmationPrompt(prompt, 2)
		if err != nil {
			return err
		}
		if proceed {
			if err := aws.NukeAllResources(account, regions); err != nil {
				return err
			}
		}
	} else {
		logging.Logger.Infoln("The --force flag is set, so waiting for 10 seconds before proceeding to nuke everything in your account. If you don't want to proceed, hit CTRL+C now!!")
		for i := 10; i > 0; i-- {
			fmt.Printf("%d...", i)
			time.Sleep(1 * time.Second)
		}

		if err := aws.NukeAllResources(account, regions); err != nil {
			return err
		}
	}

	// Remove the progressbar, now that we're ready to display the table report
	p := progressbar.GetProgressbar()
	// This next entry is necessary to workaround an issue where the spinner is not reliably cleaned up beofre the
	// final run report table is printed
	fmt.Print("\r")
	p.Stop()
	pterm.Println()

	// Print the report showing the user what happened with each resource
	ui.PrintRunReport(os.Stdout)

	return nil
}

func awsDefaults(c *cli.Context) error {
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
		return fmt.Errorf("Failed to select regions: %s", err)
	}

	if c.Bool("sg-only") {
		logging.Logger.Info("Not removing default VPCs.")
	} else {
		err = nukeDefaultVpcs(c, targetRegions)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	err = nukeDefaultSecurityGroups(c, targetRegions)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func nukeDefaultVpcs(c *cli.Context, regions []string) error {
	logging.Logger.Infof("Discovering default VPCs")
	vpcPerRegion := aws.NewVpcPerRegion(regions)
	vpcPerRegion, err := aws.GetDefaultVpcs(vpcPerRegion)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(vpcPerRegion) == 0 {
		logging.Logger.Info("No default VPCs found.")
		return nil
	}

	for _, vpc := range vpcPerRegion {
		logging.Logger.Infof("* Default VPC %s %s", vpc.VpcId, vpc.Region)
	}

	var proceed bool
	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke the default VPCs listed above? Enter 'nuke' to confirm (or exit with ^C): "
		proceed, err = confirmationPrompt(prompt, 2)
		if err != nil {
			return err
		}
	}

	if proceed || c.Bool("force") {
		err := aws.NukeVpcs(vpcPerRegion)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	return nil
}

func nukeDefaultSecurityGroups(c *cli.Context, regions []string) error {
	logging.Logger.Infof("Discovering default security groups")
	defaultSgs, err := aws.GetDefaultSecurityGroups(regions)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, sg := range defaultSgs {
		logging.Logger.Infof("* Default rules for SG %s %s %s", sg.GroupId, sg.GroupName, sg.Region)
	}

	var proceed bool
	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke the rules in these default security groups ? Enter 'nuke' to confirm (or exit with ^C): "
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
	ui.UrgentMessage("THE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

	shellOptions := shell.ShellOptions{Logger: logging.Logger}

	// retry prompt on invalid input so user can avoid rescanning all resources
	prompts := 0
	for prompts < maxPrompts {
		input, err := shell.PromptUserForInput(prompt, &shellOptions)
		if err != nil {
			return false, errors.WithStackTrace(err)
		}

		if strings.ToLower(input) == "nuke" {
			return true, nil
		}

		fmt.Printf("Invalid value '%s' was entered.\n", input)
		prompts++
	}

	return false, nil
}

func awsInspect(c *cli.Context) error {
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
		return errors.WithStackTrace(err)
	}

	query, err := aws.NewQuery(
		c.StringSlice("region"),
		c.StringSlice("exclude-region"),
		c.StringSlice("resource-type"),
		c.StringSlice("exclude-resource-type"),
		*excludeAfter,
	)
	if err != nil {
		return aws.QueryCreationError{Underlying: err}
	}

	accountResources, err := aws.InspectResources(query)
	if err != nil {
		return errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
	}

	foundResources := aws.ExtractResourcesForPrinting(accountResources)

	for _, resource := range foundResources {
		logging.Logger.Infoln(resource)
	}

	return nil
}
