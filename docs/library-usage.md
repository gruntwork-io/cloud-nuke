# Library Usage

You can import cloud-nuke into Go projects for programmatically inspecting and counting resources.

```go
package main

import (
	"context"
	"fmt"
	"time"

	nuke_aws "github.com/gruntwork-io/cloud-nuke/aws"
	nuke_config "github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/reporting"
)

func main() {
	targetRegions := []string{"us-east-1", "us-west-1", "us-west-2"}
	excludeRegions := []string{}
	resourceTypes := []string{"ec2", "vpc"}
	excludeResourceTypes := []string{}
	excludeAfter := time.Now()
	includeAfter := time.Now().AddDate(-1, 0, 0)
	timeout := time.Duration(10 * time.Second)

	// Configure query parameters for resource search
	query, err := nuke_aws.NewQuery(
		targetRegions,
		excludeRegions,
		resourceTypes,
		excludeResourceTypes,
		&excludeAfter,
		&includeAfter,
		false,    // listUnaliasedKMSKeys
		&timeout,
		false, // defaultOnly
		false, // excludeFirstSeen
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	nukeConfig := nuke_config.Config{}
	collector := reporting.NewCollector()

	accountResources, err := nuke_aws.GetAllResources(context.Background(), query, nukeConfig, collector)
	if err != nil {
		fmt.Println(err)
		return
	}

	usWest1Resources := accountResources.GetRegion("us-west-1")

	// Count resources of a specific type in a region
	fmt.Printf("EC2 count in us-west-1: %d\n", usWest1Resources.CountOfResourceType("ec2"))

	// Check if a resource type exists in a region
	fmt.Printf("EC2 present: %t\n", usWest1Resources.ResourceTypePresent("ec2"))

	// Get all resource identifiers for a type
	resourceIds := usWest1Resources.IdentifiersForResourceType("ec2")
	fmt.Printf("Resource IDs: %s\n", resourceIds)
}
```
