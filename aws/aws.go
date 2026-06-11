package aws

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/hashicorp/go-multierror"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
)

// GetAllResources - Lists all aws resources
func GetAllResources(c context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*AwsAccountResources, error) {
	configObj.AddExcludeAfterTime(query.ExcludeAfter)
	configObj.AddIncludeAfterTime(query.IncludeAfter)
	configObj.AddIncludeTags(query.IncludeTags)
	configObj.AddTimeout(query.Timeout)
	// Only override the config file value if the CLI flag is explicitly set to true
	if query.ListUnaliasedKMSKeys {
		configObj.KMSCustomerKeys.IncludeUnaliasedKeys = true
	}

	// Setting the DefaultOnly field
	// This function only sets the objects that have the `DefaultOnly` field, currently VPC, Subnet, and Security Group.
	configObj.AddEC2DefaultOnly(query.DefaultOnly)

	account := AwsAccountResources{
		Resources: make(map[string]AwsResources),
	}

	c = context.WithValue(c, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)

	parallelism := util.GetParallelism(c)

	// Build per-region sessions and registered resources up front (sequential and
	// cheap). The account ID is identical across all regions, so it is fetched
	// once here and stored on the context, avoiding a data race on the telemetry
	// package global that concurrent per-region writes would otherwise cause.
	type regionWork struct {
		region    string
		resources []*resources.AwsResource
	}
	var regionWorks []regionWork
	accountIDFetched := false
	for _, region := range query.Regions {
		cloudNukeSession, errSession := NewSession(region)
		if errSession != nil {
			return nil, errSession
		}

		if !accountIDFetched {
			if accountId, err := util.GetCurrentAccountId(cloudNukeSession); err == nil {
				telemetry.SetAccountId(accountId)
				c = context.WithValue(c, util.AccountIdKey, accountId)
				accountIDFetched = true
			}
		}

		regionWorks = append(regionWorks, regionWork{
			region:    region,
			resources: GetAndInitRegisteredResources(cloudNukeSession, region),
		})
	}

	// Scan every (region, resource type) pair concurrently under a single global
	// concurrency cap, so total in-flight API calls are bounded by `parallelism`
	// (not parallelism^2 as nested per-region/per-type limits would produce).
	type indexedResource struct {
		idx      int
		resource *resources.AwsResource
	}
	foundByRegion := make(map[string][]indexedResource)
	var foundMu sync.Mutex

	g := new(errgroup.Group)
	g.SetLimit(parallelism)

	for _, rw := range regionWorks {
		region := rw.region
		for i, resource := range rw.resources {
			if !IsNukeable((*resource).ResourceName(), query.ResourceTypes) {
				continue
			}
			idx, resource := i, resource
			g.Go(func() error {
				(*resource).GetAndSetResourceConfig(configObj)

				collector.Emit(reporting.ScanProgress{
					ResourceType: (*resource).ResourceName(),
					Region:       region,
				})

				start := time.Now()
				identifiers, err := (*resource).GetAndSetIdentifiers(c, configObj)
				if err != nil {
					logging.Errorf("Unable to retrieve %v, %v", (*resource).ResourceName(), err)

					telemetry.TrackEvent(commonTelemetry.EventContext{
						EventName: fmt.Sprintf("error:GetIdentifiers:%s", (*resource).ResourceName()),
					}, map[string]interface{}{
						"region": region,
					})

					collector.Emit(reporting.GeneralError{
						ResourceType: (*resource).ResourceName(),
						Description:  fmt.Sprintf("Unable to retrieve %s", (*resource).ResourceName()),
						Error:        err.Error(),
					})
					return nil
				}

				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("Done getting %s identifiers", (*resource).ResourceName()),
				}, map[string]interface{}{
					"recordCount": len(identifiers),
					"actionTime":  time.Since(start).Seconds(),
				})

				if len(identifiers) > 0 {
					logging.Infof("Found %d %s resources in %s", len(identifiers), (*resource).ResourceName(), region)

					foundMu.Lock()
					foundByRegion[region] = append(foundByRegion[region], indexedResource{idx: idx, resource: resource})
					foundMu.Unlock()

					for _, id := range identifiers {
						nukable, reason := true, ""
						if _, err := (*resource).IsNukable(id); err != nil {
							nukable, reason = false, err.Error()
						}
						collector.Emit(reporting.ResourceFound{
							ResourceType: (*resource).ResourceName(),
							Region:       region,
							Identifier:   id,
							Nukable:      nukable,
							Reason:       reason,
						})
					}
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Reassemble each region's resources in registry order so that nuking respects
	// dependency order (e.g. instances/ENIs/NAT before VPC).
	for region, found := range foundByRegion {
		if len(found) == 0 {
			continue
		}
		sort.Slice(found, func(a, b int) bool { return found[a].idx < found[b].idx })
		awsResources := AwsResources{Resources: make([]*resources.AwsResource, len(found))}
		for i, r := range found {
			awsResources.Resources[i] = r.resource
		}
		account.Resources[region] = awsResources
	}

	logging.Info("Done searching for resources")
	logging.Infof("Found total of %d resources", account.TotalResourceCount())

	return &account, nil
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{}
	for _, resource := range GetAllRegisteredResources() {
		resourceTypes = append(resourceTypes, (*resource).ResourceName())
	}

	sort.Strings(resourceTypes)
	return resourceTypes
}

// IsValidResourceType - Checks if a resourceType is valid or not
func IsValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}

// IsNukeable - Checks if we should nuke a resource or not
func IsNukeable(resourceType string, resourceTypes []string) bool {
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}

func nukeAllResourcesInRegion(ctx context.Context, account *AwsAccountResources, region string, collector *reporting.Collector) error {
	var allErrors *multierror.Error
	resourcesInRegion := account.Resources[region]

	for _, awsResource := range resourcesInRegion.Resources {
		length := len((*awsResource).ResourceIdentifiers())

		// Split api calls into batches
		logging.Debugf("Terminating %d awsResource in batches", length)
		batches := util.Split((*awsResource).ResourceIdentifiers(), (*awsResource).MaxBatchSize())

		for i, batch := range batches {
			// Emit progress event (CLIRenderer updates its progress bar)
			collector.Emit(reporting.NukeProgress{
				ResourceType: (*awsResource).ResourceName(),
				Region:       region,
				BatchSize:    len(batch),
			})

			results, err := (*awsResource).Nuke(ctx, batch)

			// Emit ResourceDeleted for each result
			for _, result := range results {
				errStr := ""
				if result.Error != nil {
					errStr = result.Error.Error()
				}
				collector.Emit(reporting.ResourceDeleted{
					ResourceType: (*awsResource).ResourceName(),
					Region:       region,
					Identifier:   result.Identifier,
					Success:      result.Error == nil,
					Warning:      result.Error != nil && util.IsWarningError(result.Error),
					Error:        errStr,
				})
			}

			if err != nil {
				// Handle rate limiting
				if util.IsThrottlingError(err) {
					logging.Debug(
						"Request limit reached. Waiting 1 minute before making new requests",
					)
					time.Sleep(1 * time.Minute)
					continue
				}

				allErrors = multierror.Append(allErrors, fmt.Errorf("[%s] %s: %w", region, (*awsResource).ResourceName(), err))

				// Report to telemetry - aggregated metrics of failures per resources.
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:Nuke:%s", (*awsResource).ResourceName()),
				}, map[string]interface{}{
					"region": region,
				})
			}

			if i != len(batches)-1 {
				logging.Debug("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}
		}
	}

	return allErrors.ErrorOrNil()
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(ctx context.Context, account *AwsAccountResources, regions []string, collector *reporting.Collector) error {
	// Emit NukeStarted event (CLIRenderer will initialize progress bar)
	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	parallelism := util.GetParallelism(ctx)

	var mu sync.Mutex
	var allErrors *multierror.Error

	// Regional resources are independent across regions and safe to nuke
	// concurrently. The `global` pseudo-region (IAM, Route53, CloudFront, etc.)
	// must be nuked AFTER all regional resources, because regional resources can
	// depend on global ones (e.g. an EC2 instance using a global IAM instance
	// profile, or a regional RDS global-cluster membership belonging to a global
	// cluster). Running it concurrently would risk dependency-violation errors.
	regionalRegions := make([]string, 0, len(regions))
	hasGlobal := false
	for _, region := range regions {
		if region == GlobalRegion {
			hasGlobal = true
			continue
		}
		regionalRegions = append(regionalRegions, region)
	}

	nukeRegion := func(region string) {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Creating session for region",
		}, map[string]interface{}{
			"region": region,
		})

		if err := nukeAllResourcesInRegion(ctx, account, region, collector); err != nil {
			mu.Lock()
			allErrors = multierror.Append(allErrors, err)
			mu.Unlock()
		}

		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})
	}

	g := new(errgroup.Group)
	g.SetLimit(parallelism)

	for _, region := range regionalRegions {
		region := region
		g.Go(func() error {
			nukeRegion(region)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		allErrors = multierror.Append(allErrors, err)
	}

	// Nuke global resources last, after every regional resource is gone.
	if hasGlobal {
		nukeRegion(GlobalRegion)
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}
