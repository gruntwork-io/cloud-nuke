package commands

import "github.com/urfave/cli/v2"

// Default values
const (
	DefaultOutputFormat = "table"
	DefaultDuration     = "0s"
	DefaultLogLevel     = "info"
	NukeConfirmationWord = "nuke"
	ForceNukeCountdown  = 10
	MaxConfirmationAttempts = 2
)

// Flag Names
// These constants define all CLI flag names to avoid typos and enable refactoring
const (
	FlagListResourceTypes      = "list-resource-types"
	FlagLogLevel               = "log-level"
	FlagConfig                 = "config"
	FlagOlderThan              = "older-than"
	FlagNewerThan              = "newer-than"
	FlagTimeout                = "timeout"
	FlagDryRun                 = "dry-run"
	FlagForce                  = "force"
	FlagOutputFormat           = "output-format"
	FlagOutputFile             = "output-file"
	FlagDeleteUnaliasedKMSKeys = "delete-unaliased-kms-keys"
	FlagListUnaliasedKMSKeys   = "list-unaliased-kms-keys"
	FlagExcludeFirstSeen       = "exclude-first-seen"
	FlagSGOnly                 = "sg-only"
	FlagProjectID              = "project-id"
	FlagResourceType           = "resource-type"
	FlagExcludeResourceType    = "exclude-resource-type"
	FlagRegion                 = "region"
	FlagExcludeRegion          = "exclude-region"
)

// Common flag sets for reuse across commands

// CommonTimeFlags returns flags for time-based filtering
func CommonTimeFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  FlagOlderThan,
			Usage: "Only delete resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
			Value: DefaultDuration,
		},
		&cli.StringFlag{
			Name:  FlagNewerThan,
			Usage: "Only delete resources newer than this specified value. Can be any valid Go duration, such as 10m or 8h.",
			Value: DefaultDuration,
		},
	}
}

// CommonResourceTypeFlags returns flags for resource type filtering
func CommonResourceTypeFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  FlagResourceType,
			Usage: "Resource types to nuke. Include multiple times if more than one.",
		},
		&cli.StringSliceFlag{
			Name:  FlagExcludeResourceType,
			Usage: "Resource types to exclude from nuking. Include multiple times if more than one.",
		},
		&cli.BoolFlag{
			Name:  FlagListResourceTypes,
			Usage: "List available resource types",
		},
	}
}

// InspectResourceTypeFlags returns flags for resource type filtering in inspect commands
func InspectResourceTypeFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  FlagResourceType,
			Usage: "Resource types to inspect. Include multiple times if more than one.",
		},
		&cli.StringSliceFlag{
			Name:  FlagExcludeResourceType,
			Usage: "Resource types to exclude from inspection. Include multiple times if more than one.",
		},
		&cli.BoolFlag{
			Name:  FlagListResourceTypes,
			Usage: "List available resource types",
		},
	}
}

// CommonExecutionFlags returns flags for execution control
func CommonExecutionFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  FlagDryRun,
			Usage: "Dry run without taking any action.",
		},
		&cli.BoolFlag{
			Name:  FlagForce,
			Usage: "Skip nuke confirmation prompt. WARNING: this will automatically delete all targeted resources without any confirmation.",
		},
		&cli.StringFlag{
			Name:  FlagTimeout,
			Usage: "Resource execution timeout.",
		},
	}
}

// CommonOutputFlags returns flags for output formatting
func CommonOutputFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  FlagOutputFormat,
			Usage: "Output format (table, json)",
			Value: DefaultOutputFormat,
		},
		&cli.StringFlag{
			Name:  FlagOutputFile,
			Usage: "Write output to file instead of stdout (optional)",
		},
		&cli.StringFlag{
			Name:    FlagLogLevel,
			Value:   DefaultLogLevel,
			Usage:   "Set log level",
			EnvVars: []string{"LOG_LEVEL"},
		},
	}
}

// ConfigFlag returns the config file flag
func ConfigFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  FlagConfig,
		Usage: "YAML file specifying matching rules.",
	}
}

// RegionFlags returns region-related flags (applicable to both AWS and GCP)
func RegionFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  FlagRegion,
			Usage: "Regions to include. Include multiple times if more than one.",
		},
		&cli.StringSliceFlag{
			Name:  FlagExcludeRegion,
			Usage: "Regions to exclude. Include multiple times if more than one.",
		},
	}
}

// GCPProjectFlag returns the GCP project ID flag
func GCPProjectFlag() cli.Flag {
	return &cli.StringFlag{
		Name:     FlagProjectID,
		Usage:    "GCP Project ID to nuke resources from.",
		Required: true,
	}
}

// CombineFlags combines multiple flag slices into one
func CombineFlags(flagSets ...[]cli.Flag) []cli.Flag {
	var combined []cli.Flag
	for _, flags := range flagSets {
		combined = append(combined, flags...)
	}
	return combined
}
