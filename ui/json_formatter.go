package ui

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// GetOutputWriter returns a writer for the specified output file or stdout if empty
func GetOutputWriter(outputFile string) (io.Writer, func() error, error) {
	if outputFile == "" {
		// Return stdout with a no-op closer
		return os.Stdout, func() error { return nil }, nil
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, nil, errors.WithStackTrace(err)
	}

	return file, file.Close, nil
}

// RenderInspectAsJSON renders AWS inspection results as JSON
func RenderInspectAsJSON(account *aws.AwsAccountResources, query *aws.Query, w io.Writer) error {
	output := buildInspectOutput(account, query)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// RenderGcpInspectAsJSON renders GCP inspection results as JSON
func RenderGcpInspectAsJSON(project *gcp.GcpProjectResources, w io.Writer) error {
	output := buildGcpInspectOutput(project)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// RenderNukeReportAsJSON renders nuke operation results as JSON
//
// Deprecated: Use the reporting package with Collector/Renderer pattern instead.
// This function reads from the deprecated report package's global state.
// See renderers.NukeJSONRenderer for the new approach.
func RenderNukeReportAsJSON(w io.Writer) error {
	records := report.GetRecords()
	generalErrors := report.GetErrors()
	output := buildNukeOutput(records, generalErrors)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// buildInspectOutput constructs the JSON output structure for AWS inspection
func buildInspectOutput(account *aws.AwsAccountResources, query *aws.Query) InspectOutput {
	var resources []ResourceInfo
	byType := make(map[string]int)
	byRegion := make(map[string]int)
	nukableCount := 0
	nonNukableCount := 0

	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			resourceName := (*foundResources).ResourceName()
			for _, identifier := range (*foundResources).ResourceIdentifiers() {
				nukable := true
				nukableReason := ""

				_, err := (*foundResources).IsNukable(identifier)
				if err != nil {
					nukable = false
					nukableReason = err.Error()
					nonNukableCount++
				} else {
					nukableCount++
				}

				resources = append(resources, ResourceInfo{
					ResourceType:  resourceName,
					Region:        region,
					Identifier:    identifier,
					Nukable:       nukable,
					NukableReason: nukableReason,
				})

				byType[resourceName]++
				byRegion[region]++
			}
		}
	}

	queryParams := QueryParams{
		Regions:              query.Regions,
		ResourceTypes:        query.ResourceTypes,
		ExcludeAfter:         query.ExcludeAfter,
		IncludeAfter:         query.IncludeAfter,
		ListUnaliasedKMSKeys: query.ListUnaliasedKMSKeys,
	}

	return InspectOutput{
		Timestamp: time.Now(),
		Command:   "inspect-aws",
		Query:     queryParams,
		Resources: resources,
		Summary: InspectSummary{
			TotalResources: len(resources),
			Nukable:        nukableCount,
			NonNukable:     nonNukableCount,
			ByType:         byType,
			ByRegion:       byRegion,
		},
	}
}

// buildGcpInspectOutput constructs the JSON output structure for GCP inspection
func buildGcpInspectOutput(project *gcp.GcpProjectResources) InspectOutput {
	var resources []ResourceInfo
	byType := make(map[string]int)
	byRegion := make(map[string]int)
	nukableCount := 0
	nonNukableCount := 0

	for region, resourcesInRegion := range project.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			resourceName := (*foundResources).ResourceName()
			for _, identifier := range (*foundResources).ResourceIdentifiers() {
				nukable := true
				nukableReason := ""

				_, err := (*foundResources).IsNukable(identifier)
				if err != nil {
					nukable = false
					nukableReason = err.Error()
					nonNukableCount++
				} else {
					nukableCount++
				}

				resources = append(resources, ResourceInfo{
					ResourceType:  resourceName,
					Region:        region,
					Identifier:    identifier,
					Nukable:       nukable,
					NukableReason: nukableReason,
				})

				byType[resourceName]++
				byRegion[region]++
			}
		}
	}

	return InspectOutput{
		Timestamp: time.Now(),
		Command:   "inspect-gcp",
		Resources: resources,
		Summary: InspectSummary{
			TotalResources: len(resources),
			Nukable:        nukableCount,
			NonNukable:     nonNukableCount,
			ByType:         byType,
			ByRegion:       byRegion,
		},
	}
}

// buildNukeOutput constructs the JSON output structure for nuke operations
func buildNukeOutput(records map[string]report.Entry, generalErrors map[string]report.GeneralError) NukeOutput {
	resources := []NukeResourceInfo{}
	deletedCount := 0
	failedCount := 0

	for _, entry := range records {
		status := "deleted"
		errorMsg := ""
		if entry.Error != nil {
			status = "failed"
			errorMsg = entry.Error.Error()
			failedCount++
		} else {
			deletedCount++
		}

		resources = append(resources, NukeResourceInfo{
			Identifier:   entry.Identifier,
			ResourceType: entry.ResourceType,
			Status:       status,
			Error:        errorMsg,
		})
	}

	errors := []GeneralErrorInfo{}
	for _, generalErr := range generalErrors {
		errors = append(errors, GeneralErrorInfo{
			ResourceType: generalErr.ResourceType,
			Description:  generalErr.Description,
			Error:        generalErr.Error.Error(),
		})
	}

	return NukeOutput{
		Timestamp: time.Now(),
		Command:   "nuke",
		Resources: resources,
		Errors:    errors,
		Summary: NukeSummary{
			Total:         len(resources),
			Deleted:       deletedCount,
			Failed:        failedCount,
			GeneralErrors: len(errors),
		},
	}
}

// ShouldSuppressProgressOutput returns true if progress output should be suppressed
func ShouldSuppressProgressOutput(outputFormat string) bool {
	return outputFormat == "json"
}
