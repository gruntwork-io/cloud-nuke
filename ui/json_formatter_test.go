package ui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOutputWriter(t *testing.T) {
	t.Run("returns stdout when empty string", func(t *testing.T) {
		writer, closer, err := GetOutputWriter("")
		require.NoError(t, err)
		assert.NotNil(t, writer)
		assert.NotNil(t, closer)
		assert.Equal(t, os.Stdout, writer)
		err = closer()
		assert.NoError(t, err)
	})

	t.Run("creates file when path provided", func(t *testing.T) {
		tempFile := "/tmp/test-output-" + time.Now().Format("20060102150405") + ".json"
		writer, closer, err := GetOutputWriter(tempFile)
		require.NoError(t, err)
		assert.NotNil(t, writer)
		assert.NotNil(t, closer)

		// Write some data
		_, err = writer.Write([]byte("test data"))
		assert.NoError(t, err)

		// Close the file
		err = closer()
		assert.NoError(t, err)

		// Verify file exists and has content
		content, err := os.ReadFile(tempFile)
		assert.NoError(t, err)
		assert.Equal(t, "test data", string(content))

		// Clean up
		os.Remove(tempFile)
	})
}

func TestShouldSuppressProgressOutput(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected bool
	}{
		{"json format", "json", true},
		{"table format", "table", false},
		{"empty format", "", false},
		{"unknown format", "xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSuppressProgressOutput(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildNukeOutput(t *testing.T) {
	// Set up test data
	records := map[string]report.Entry{
		"resource1": {
			Identifier:   "i-123456",
			ResourceType: "EC2Instance",
			Error:        nil,
		},
		"resource2": {
			Identifier:   "sg-789012",
			ResourceType: "SecurityGroup",
			Error:        &testError{msg: "Cannot delete"},
		},
	}

	generalErrors := map[string]report.GeneralError{
		"error1": {
			ResourceType: "S3Bucket",
			Description:  "Failed to list",
			Error:        &testError{msg: "Access denied"},
		},
	}

	output := buildNukeOutput(records, generalErrors)

	// Verify the structure
	assert.Equal(t, "nuke", output.Command)
	assert.Equal(t, 2, output.Summary.Total)
	assert.Equal(t, 1, output.Summary.Deleted)
	assert.Equal(t, 1, output.Summary.Failed)
	assert.Equal(t, 1, output.Summary.GeneralErrors)
	assert.Equal(t, 2, len(output.Resources))
	assert.Equal(t, 1, len(output.Errors))

	// Verify resources
	resourceMap := make(map[string]NukeResourceInfo)
	for _, r := range output.Resources {
		resourceMap[r.Identifier] = r
	}

	assert.Equal(t, "deleted", resourceMap["i-123456"].Status)
	assert.Equal(t, "", resourceMap["i-123456"].Error)
	assert.Equal(t, "failed", resourceMap["sg-789012"].Status)
	assert.Equal(t, "Cannot delete", resourceMap["sg-789012"].Error)
}

func TestRenderNukeReportAsJSON(t *testing.T) {
	// Reset and set up test data
	report.ResetRecords()
	report.ResetErrors()
	defer func() {
		report.ResetRecords()
		report.ResetErrors()
	}()

	// Add test records
	report.Record(report.Entry{
		Identifier:   "vpc-123",
		ResourceType: "VPC",
		Error:        nil,
	})
	report.Record(report.Entry{
		Identifier:   "igw-456",
		ResourceType: "InternetGateway",
		Error:        &testError{msg: "Dependency error"},
	})

	// Add general error
	report.RecordError(report.GeneralError{
		ResourceType: "EC2",
		Description:  "API rate limit",
		Error:        &testError{msg: "Too many requests"},
	})

	// Test JSON rendering
	var buf bytes.Buffer
	err := RenderNukeReportAsJSON(&buf)
	require.NoError(t, err)

	// Parse and verify JSON
	var output NukeOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "nuke", output.Command)
	assert.Equal(t, 2, output.Summary.Total)
	assert.Equal(t, 1, output.Summary.Deleted)
	assert.Equal(t, 1, output.Summary.Failed)
	assert.Equal(t, 1, len(output.Errors))
}

func TestJSONOutputValidity(t *testing.T) {
	// Test that all JSON outputs produce valid JSON
	t.Run("NukeOutput JSON validity", func(t *testing.T) {
		output := NukeOutput{
			Timestamp: time.Now(),
			Command:   "nuke",
			Account:   "123456789012",
			Regions:   []string{"us-east-1", "us-west-2"},
			Resources: []NukeResourceInfo{
				{
					Identifier:   "test-resource",
					ResourceType: "TestType",
					Status:       "deleted",
					Error:        "",
				},
			},
			Errors: []GeneralErrorInfo{
				{
					ResourceType: "TestType",
					Description:  "Test error",
					Error:        "Error message",
				},
			},
			Summary: NukeSummary{
				Total:         1,
				Deleted:       1,
				Failed:        0,
				GeneralErrors: 1,
			},
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(output)
		require.NoError(t, err)

		// Verify it's valid JSON by unmarshaling
		var parsed NukeOutput
		err = json.Unmarshal(jsonBytes, &parsed)
		require.NoError(t, err)

		// Verify key fields
		assert.Equal(t, output.Command, parsed.Command)
		assert.Equal(t, output.Account, parsed.Account)
		assert.Equal(t, len(output.Resources), len(parsed.Resources))
		assert.Equal(t, len(output.Errors), len(parsed.Errors))
	})

	t.Run("InspectOutput JSON validity", func(t *testing.T) {
		excludeTime := time.Now().Add(-24 * time.Hour)
		output := InspectOutput{
			Timestamp: time.Now(),
			Command:   "inspect-aws",
			Query: QueryParams{
				Regions:              []string{"us-east-1"},
				ResourceTypes:        []string{"ec2", "s3"},
				ExcludeAfter:         &excludeTime,
				IncludeAfter:         nil,
				ListUnaliasedKMSKeys: false,
			},
			Resources: []ResourceInfo{
				{
					ResourceType:  "ec2",
					Region:        "us-east-1",
					Identifier:    "i-123456",
					Nukable:       true,
					NukableReason: "",
					Tags: map[string]string{
						"Name": "test-instance",
					},
				},
				{
					ResourceType:  "s3",
					Region:        "us-east-1",
					Identifier:    "test-bucket",
					Nukable:       false,
					NukableReason: "Bucket not empty",
				},
			},
			Summary: InspectSummary{
				TotalResources: 2,
				Nukable:        1,
				NonNukable:     1,
				ByType: map[string]int{
					"ec2": 1,
					"s3":  1,
				},
				ByRegion: map[string]int{
					"us-east-1": 2,
				},
			},
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(output)
		require.NoError(t, err)

		// Verify it's valid JSON by unmarshaling
		var parsed InspectOutput
		err = json.Unmarshal(jsonBytes, &parsed)
		require.NoError(t, err)

		// Verify key fields
		assert.Equal(t, output.Command, parsed.Command)
		assert.Equal(t, len(output.Resources), len(parsed.Resources))
		assert.Equal(t, output.Summary.TotalResources, parsed.Summary.TotalResources)
		assert.Equal(t, output.Summary.Nukable, parsed.Summary.Nukable)
	})
}

// Test that buildInspectOutput creates correct structure
// Note: Full integration testing with AWS resources requires mocking
// the complex AwsResource interface, so we focus on testing the
// output structure and JSON validity in other tests

// Test helper - simple error implementation
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}