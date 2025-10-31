package commands

import (
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/urfave/cli/v2"
)

// loadConfigFile loads and parses a config file from the given path
func loadConfigFile(configFilePath string) (config.Config, error) {
	if configFilePath == "" {
		return config.Config{}, nil
	}

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: telemetry.EventReadingConfig,
	}, map[string]interface{}{})

	configObjPtr, err := config.GetConfig(configFilePath)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: telemetry.EventErrorReadingConfig,
		}, map[string]interface{}{})
		return config.Config{}, ConfigFileReadError{FilePath: configFilePath, Underlying: err}
	}

	return *configObjPtr, nil
}

// parseAndApplyTimeFilters parses time filter flags and applies them to the config
func parseAndApplyTimeFilters(c *cli.Context, configObj *config.Config) error {
	excludeAfter, err := parseDurationParam(FlagOlderThan, c.String(FlagOlderThan))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	includeAfter, err := parseDurationParam(FlagNewerThan, c.String(FlagNewerThan))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Apply time filters to config
	configObj.ApplyTimeFilters(excludeAfter, includeAfter)

	return nil
}

// parseAndApplyTimeout parses the timeout flag and applies it to the config
func parseAndApplyTimeout(c *cli.Context, configObj *config.Config) error {
	timeout, err := parseTimeoutDurationParam(FlagTimeout, c.String(FlagTimeout))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if timeout != nil {
		configObj.AddTimeout(timeout)
	}

	return nil
}

// confirmNuke handles the nuke confirmation prompt and countdown
// Returns true if the nuke should proceed, false otherwise
func confirmNuke(c *cli.Context, hasResources bool) (bool, error) {
	if !hasResources {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: telemetry.EventNoResources,
		}, map[string]interface{}{})
		logging.Info("Nothing to nuke, you're all good!")
		return false, nil
	}

	if c.Bool(FlagDryRun) {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: telemetry.EventSkippingDryRun,
		}, map[string]interface{}{})
		logging.Info("Not taking any action as dry-run set to true.")
		return false, nil
	}

	if !c.Bool(FlagForce) {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: telemetry.EventAwaitingConfirmation,
		}, map[string]interface{}{})

		promptMessage := fmt.Sprintf("\nAre you sure you want to nuke all listed resources? Enter '%s' to confirm (or exit with ^C) ",
			NukeConfirmationWord)

		proceed, err := ui.RenderNukeConfirmationPrompt(promptMessage, MaxConfirmationAttempts)
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: telemetry.EventErrorConfirming,
			}, map[string]interface{}{})
			return false, err
		}

		if !proceed {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: telemetry.EventUserAborted,
			}, map[string]interface{}{})
			return false, nil
		}

		return true, nil
	}

	// Force flag is set
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: telemetry.EventForcingNuke,
	}, map[string]interface{}{})

	warningMessage := fmt.Sprintf("The --force flag is set, so waiting for %d seconds before proceeding to nuke everything. If you don't want to proceed, hit CTRL+C now!!",
		ForceNukeCountdown)
	logging.Info(warningMessage)

	for i := ForceNukeCountdown; i > 0; i-- {
		fmt.Printf("%d...", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Println()

	return true, nil
}

// parseLogLevel parses and sets the log level from CLI context
func parseLogLevel(c *cli.Context) error {
	logLevel := c.String(FlagLogLevel)
	parseErr := logging.ParseLogLevel(logLevel)
	if parseErr != nil {
		return errors.WithStackTrace(InvalidLogLevelError{
			Value:      logLevel,
			Underlying: parseErr,
		})
	}
	return nil
}

// parseDurationParam parses a duration string (e.g., "10h", "5d") and converts it to a time.Time
// representing a point in the past. This is used for --older-than and --newer-than flags.
// Returns nil if the paramValue is empty or the default value.
func parseDurationParam(flagName string, paramValue string) (*time.Time, error) {
	if paramValue == DefaultDuration || paramValue == "" {
		return nil, nil //nolint:nilnil // Returning (nil, nil) is semantically correct here - means "no value provided, not an error"
	}

	// Parse the duration string (e.g., "10h" -> 10 hours)
	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(InvalidDurationError{
			FlagName:   flagName,
			Value:      paramValue,
			Underlying: err,
		})
	}

	// Make it negative so it goes back in time
	duration = -1 * duration

	// Calculate the time point in the past
	excludeAfter := time.Now().Add(duration)
	return &excludeAfter, nil
}

// parseTimeoutDurationParam parses a timeout duration string (e.g., "30m", "1h").
// Returns nil if the paramValue is empty or the default value.
func parseTimeoutDurationParam(flagName string, paramValue string) (*time.Duration, error) {
	if paramValue == DefaultDuration || paramValue == "" {
		return nil, nil //nolint:nilnil // Returning (nil, nil) is semantically correct here - means "no value provided, not an error"
	}

	// Parse the duration string
	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(InvalidDurationError{
			FlagName:   flagName,
			Value:      paramValue,
			Underlying: err,
		})
	}
	return &duration, nil
}
