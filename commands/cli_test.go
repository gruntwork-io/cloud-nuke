package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	goCommonErrors "github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestParseDuration(t *testing.T) {
	now := time.Now()
	then, err := parseDurationParam("test-duration", "1h")
	if err != nil {
		assert.Fail(t, goCommonErrors.WithStackTrace(err).Error())
	}

	if now.Hour() == 0 {
		// At midnight, now.Hour returns 0 so we need to handle that specially.
		assert.Equal(t, 23, then.Hour())
		// Also, the date changed, so 1 hour ago will be the previous day.
		assert.Equal(t, now.Day()-1, then.Day())
	} else {
		assert.Equal(t, now.Hour()-1, then.Hour())
		assert.Equal(t, now.Day(), then.Day())
	}

	assert.Equal(t, now.Month(), then.Month())
	assert.Equal(t, now.Year(), then.Year())
}

func TestParseDurationInvalidFormat(t *testing.T) {
	value, err := parseDurationParam("test-duration", "")
	assert.NoError(t, err)
	assert.Nil(t, value)
}

func TestStructuredErrors(t *testing.T) {
	t.Run("InvalidDurationError", func(t *testing.T) {
		_, err := parseDurationParam("older-than", "invalid-duration")
		assert.Error(t, err)

		// Unwrap the stacktrace to get the actual error
		var durationErr InvalidDurationError
		if errors.As(err, &durationErr) {
			assert.Equal(t, "older-than", durationErr.FlagName)
			assert.Equal(t, "invalid-duration", durationErr.Value)
			assert.NotNil(t, durationErr.Underlying)
			assert.Contains(t, durationErr.Error(), "older-than")
			assert.Contains(t, durationErr.Error(), "invalid-duration")
		} else {
			t.Error("Expected InvalidDurationError")
		}
	})

	t.Run("InvalidDurationError for timeout", func(t *testing.T) {
		_, err := parseTimeoutDurationParam("timeout", "not-a-duration")
		assert.Error(t, err)

		var durationErr InvalidDurationError
		if errors.As(err, &durationErr) {
			assert.Equal(t, "timeout", durationErr.FlagName)
			assert.Equal(t, "not-a-duration", durationErr.Value)
			assert.Contains(t, durationErr.Error(), "timeout")
		} else {
			t.Error("Expected InvalidDurationError")
		}
	})

	t.Run("ConfigFileReadError", func(t *testing.T) {
		err := ConfigFileReadError{
			FilePath:   "/path/to/config.yaml",
			Underlying: errors.New("file not found"),
		}
		assert.Contains(t, err.Error(), "/path/to/config.yaml")
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("InvalidLogLevelError", func(t *testing.T) {
		err := InvalidLogLevelError{
			Value:      "invalid-level",
			Underlying: errors.New("not a valid log level"),
		}
		assert.Contains(t, err.Error(), "invalid-level")
		assert.Contains(t, err.Error(), "not a valid log level")
	})

	t.Run("InvalidFlagError", func(t *testing.T) {
		err := InvalidFlagError{
			Name:  "test-flag",
			Value: "bad-value",
		}
		assert.Contains(t, err.Error(), "test-flag")
		assert.Contains(t, err.Error(), "bad-value")
	})
}

func TestListResourceTypes(t *testing.T) {
	allAWSResourceTypes := aws.ListResourceTypes()
	assert.Greater(t, len(allAWSResourceTypes), 0)
	assert.Contains(t, allAWSResourceTypes, (&resources.EC2Instances{}).ResourceName())
}

func TestIsValidResourceType(t *testing.T) {
	allAWSResourceTypes := aws.ListResourceTypes()
	ec2ResourceName := (*&resources.EC2Instances{}).ResourceName()
	assert.Equal(t, aws.IsValidResourceType(ec2ResourceName, allAWSResourceTypes), true)
	assert.Equal(t, aws.IsValidResourceType("xyz", allAWSResourceTypes), false)
}

func TestIsNukeable(t *testing.T) {
	ec2ResourceName := (&resources.EC2Instances{}).ResourceName()
	amiResourceName := resources.NewAMIs().ResourceName()

	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{ec2ResourceName}), true)
	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{"all"}), true)
	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{}), true)
	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{amiResourceName}), false)
}

func TestCLIFlags(t *testing.T) {
	app := CreateCli("test-version")

	t.Run("aws command has output format flags", func(t *testing.T) {
		awsCmd := findCommand(app.Commands, "aws")
		require.NotNil(t, awsCmd)

		// Check for output-format flag
		outputFormatFlag := findFlag(awsCmd.Flags, "output-format")
		assert.NotNil(t, outputFormatFlag)
		if stringFlag, ok := outputFormatFlag.(*cli.StringFlag); ok {
			assert.Equal(t, "table", stringFlag.Value)
			assert.Contains(t, stringFlag.Usage, "Output format")
		}

		// Check for output-file flag
		outputFileFlag := findFlag(awsCmd.Flags, "output-file")
		assert.NotNil(t, outputFileFlag)
		if stringFlag, ok := outputFileFlag.(*cli.StringFlag); ok {
			assert.Contains(t, stringFlag.Usage, "Write output to file")
		}
	})

	t.Run("inspect-aws command has output format flags", func(t *testing.T) {
		inspectCmd := findCommand(app.Commands, "inspect-aws")
		require.NotNil(t, inspectCmd)

		// Check for output-format flag
		outputFormatFlag := findFlag(inspectCmd.Flags, "output-format")
		assert.NotNil(t, outputFormatFlag)
		if stringFlag, ok := outputFormatFlag.(*cli.StringFlag); ok {
			assert.Equal(t, "table", stringFlag.Value)
		}

		// Check for output-file flag
		outputFileFlag := findFlag(inspectCmd.Flags, "output-file")
		assert.NotNil(t, outputFileFlag)
	})

	t.Run("gcp command has output format flags", func(t *testing.T) {
		gcpCmd := findCommand(app.Commands, "gcp")
		require.NotNil(t, gcpCmd)

		// Check for output-format flag
		outputFormatFlag := findFlag(gcpCmd.Flags, "output-format")
		assert.NotNil(t, outputFormatFlag)

		// Check for output-file flag
		outputFileFlag := findFlag(gcpCmd.Flags, "output-file")
		assert.NotNil(t, outputFileFlag)
	})

	t.Run("inspect-gcp command has output format flags", func(t *testing.T) {
		inspectGcpCmd := findCommand(app.Commands, "inspect-gcp")
		require.NotNil(t, inspectGcpCmd)

		// Check for output-format flag
		outputFormatFlag := findFlag(inspectGcpCmd.Flags, "output-format")
		assert.NotNil(t, outputFormatFlag)

		// Check for output-file flag
		outputFileFlag := findFlag(inspectGcpCmd.Flags, "output-file")
		assert.NotNil(t, outputFileFlag)
	})
}

func TestOutputFormatValues(t *testing.T) {
	app := CreateCli("test-version")

	// Test that all commands with output-format flag have correct default
	commandsToTest := []string{"aws", "gcp", "inspect-aws", "inspect-gcp"}

	for _, cmdName := range commandsToTest {
		t.Run(cmdName+" has correct default output format", func(t *testing.T) {
			cmd := findCommand(app.Commands, cmdName)
			require.NotNil(t, cmd)

			flag := findFlag(cmd.Flags, "output-format")
			require.NotNil(t, flag)

			if stringFlag, ok := flag.(*cli.StringFlag); ok {
				assert.Equal(t, "table", stringFlag.Value, "Default should be 'table' for backward compatibility")
			}
		})
	}
}

func TestOutputFileFlag(t *testing.T) {
	app := CreateCli("test-version")

	// Test that output-file flag is optional (no default value)
	commandsToTest := []string{"aws", "gcp", "inspect-aws", "inspect-gcp"}

	for _, cmdName := range commandsToTest {
		t.Run(cmdName+" has optional output-file flag", func(t *testing.T) {
			cmd := findCommand(app.Commands, cmdName)
			require.NotNil(t, cmd)

			flag := findFlag(cmd.Flags, "output-file")
			require.NotNil(t, flag)

			if stringFlag, ok := flag.(*cli.StringFlag); ok {
				assert.Equal(t, "", stringFlag.Value, "output-file should have no default value")
				assert.Contains(t, strings.ToLower(stringFlag.Usage), "optional")
			}
		})
	}
}

// Integration test for output file creation
func TestOutputFileCreation(t *testing.T) {
	// This test would require mocking AWS calls, so we just test the file creation logic
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "test-output.json")

	// Test that the file path is valid
	assert.NotContains(t, outputFile, " ", "Path should not contain spaces")
	assert.True(t, strings.HasSuffix(outputFile, ".json"))

	// Test that we can create a file at this path
	file, err := os.Create(outputFile)
	require.NoError(t, err)
	defer file.Close()

	// Write test content
	testContent := `{"test": "data"}`
	_, err = file.WriteString(testContent)
	require.NoError(t, err)

	// Verify file exists and has content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// Helper functions
func findCommand(commands []*cli.Command, name string) *cli.Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func findFlag(flags []cli.Flag, name string) cli.Flag {
	for _, flag := range flags {
		// Get the flag names
		if stringFlag, ok := flag.(*cli.StringFlag); ok && stringFlag.Name == name {
			return flag
		}
		if boolFlag, ok := flag.(*cli.BoolFlag); ok && boolFlag.Name == name {
			return flag
		}
		if stringSliceFlag, ok := flag.(*cli.StringSliceFlag); ok && stringSliceFlag.Name == name {
			return flag
		}
	}
	return nil
}
