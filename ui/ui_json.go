package ui

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/go-commons/errors"
)

type JSONResource struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	Identifier   string `json:"identifier"`
	Nukable      bool   `json:"nukable"`
	Error        string `json:"error,omitempty"`
}

type JSONOutput struct {
	Resources []JSONResource `json:"resources"`
	Summary   JSONSummary    `json:"summary"`
}

type JSONSummary struct {
	TotalResources int `json:"total"`
	NukableCount   int `json:"nukables"`
	ErrorCount     int `json:"errors"`
}

func RenderResourcesAsJSON(account *aws.AwsAccountResources, outputFile string) error {
	var (
		resources    []JSONResource
		nukableCount int
		errorCount   int
	)

	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			for _, identifier := range (*foundResources).ResourceIdentifiers() {
				resource := JSONResource{
					ResourceType: (*foundResources).ResourceName(),
					Region:       region,
					Identifier:   identifier,
				}

				_, err := (*foundResources).IsNukable(identifier)
				if err != nil {
					resource.Nukable = false
					resource.Error = err.Error()
					errorCount++
				} else {
					resource.Nukable = true
					nukableCount++
				}

				resources = append(resources, resource)
			}
		}
	}

	output := JSONOutput{
		Resources: resources,
		Summary: JSONSummary{
			TotalResources: len(resources),
			NukableCount:   nukableCount,
			ErrorCount:     errorCount,
		},
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if outputFile != "" {
		err = os.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		fmt.Printf("JSON output written to: %s\n", outputFile)
		return nil
	}

	// print the output in terminal
	fmt.Println(string(jsonData))

	return nil
}
