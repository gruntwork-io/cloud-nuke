package ui

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/stretchr/testify/require"
)

func TestJSONStructures(t *testing.T) {
	resources := []JSONResource{
		{
			ResourceType: "ec2",
			Region:       "us-east-1",
			Identifier:   "i-1234567890abcdef0",
			Nukable:      true,
		},
		{
			ResourceType: "ec2",
			Region:       "us-east-1",
			Identifier:   "i-0987654321fedcba0",
			Nukable:      true,
		},
		{
			ResourceType: "s3",
			Region:       "us-east-1",
			Identifier:   "my-test-bucket",
			Nukable:      false,
			Error:        "Bucket not empty",
		},
	}

	output := JSONOutput{
		Resources: resources,
		Summary: JSONSummary{
			TotalResources: 3,
			NukableCount:   2,
			ErrorCount:     1,
		},
	}

	// Verify the output structure
	require.Len(t, output.Resources, 3)
	require.Equal(t, 3, output.Summary.TotalResources)
	require.Equal(t, 2, output.Summary.NukableCount)
	require.Equal(t, 1, output.Summary.ErrorCount)

	// Verify JSON marshaling works
	jsonData, err := json.MarshalIndent(output, "", "  ")
	require.NoError(t, err)
	require.Contains(t, string(jsonData), "ec2")
	require.Contains(t, string(jsonData), "s3")
	require.Contains(t, string(jsonData), "us-east-1")
	require.Contains(t, string(jsonData), "Bucket not empty")

	var parsed JSONOutput
	err = json.Unmarshal(jsonData, &parsed)

	require.NoError(t, err)
	require.Equal(t, output, parsed)
}

func TestRenderResourcesAsJSONToFile(t *testing.T) {

	tempFile := "/tmp/test-output.json"
	defer os.Remove(tempFile)

	account := &aws.AwsAccountResources{
		Resources: map[string]aws.AwsResources{},
	}

	err := RenderResourcesAsJSON(account, tempFile)
	require.NoError(t, err)

	data, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	var output JSONOutput
	err = json.Unmarshal(data, &output)

	require.NoError(t, err)
	require.Equal(t, 0, output.Summary.TotalResources)
}
