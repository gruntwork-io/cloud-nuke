package aws

import (
	"cmp"
	"context"
	"fmt"
	"slices"
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
	// Inject parallelism from the query so downstream callers (e.g. batch_deleter) can read it
	// via util.GetParallelism without needing the caller to set it on the context.
	c = context.WithValue(c, util.ParallelismKey, query.Parallelism)
	parallelism := util.GetParallelism(c)

	type indexedResource struct {
		idx      int
		resource *resources.AwsResource
	}
	type regionSetup struct {
		regionCtx context.Context
		nukeable  []indexedResource
	}

	// Phase 1: set up sessions and init resources for each region concurrently.
	// GetAndInitRegisteredResources only allocates SDK clients (no API calls), so
	// no concurrency limit is needed here.
	setups := make(map[string]*regionSetup, len(query.Regions))
	var setupMu sync.Mutex
	var setAccountOnce sync.Once

	setupGroup := new(errgroup.Group)
	for _, region := range query.Regions {
		setupGroup.Go(func() error {
			cloudNukeSession, err := NewSession(region)
			if err != nil {
				return err
			}
			regionCtx := c
			accountId, err := util.GetCurrentAccountId(cloudNukeSession)
			if err == nil {
				// All regions share the same account ID; set it once to avoid a data race.
				setAccountOnce.Do(func() { telemetry.SetAccountId(accountId) })
				regionCtx = context.WithValue(c, util.AccountIdKey, accountId)
			}
			registeredResources := GetAndInitRegisteredResources(cloudNukeSession, region)
			setup := &regionSetup{regionCtx: regionCtx}
			for i, res := range registeredResources {
				if IsNukeable((*res).ResourceName(), query.ResourceTypes) {
					setup.nukeable = append(setup.nukeable, indexedResource{idx: i, resource: res})
				}
			}
			setupMu.Lock()
			setups[region] = setup
			setupMu.Unlock()
			return nil
		})
	}
	if err := setupGroup.Wait(); err != nil {
		return nil, err
	}

	// Phase 2: scan all nukeable resources across all regions with a single shared
	// concurrency limit. Using one errgroup (instead of nested per-region groups)
	// ensures --parallelism N means at most N simultaneous API calls total.
	type resourceTask struct {
		region    string
		regionCtx context.Context
		idx       int
		resource  *resources.AwsResource
	}
	var allTasks []resourceTask
	for _, region := range query.Regions {
		setup, ok := setups[region]
		if !ok {
			continue
		}
		for _, r := range setup.nukeable {
			allTasks = append(allTasks, resourceTask{
				region:    region,
				regionCtx: setup.regionCtx,
				idx:       r.idx,
				resource:  r.resource,
			})
		}
	}

	foundByRegion := make(map[string][]indexedResource)
	var foundMu sync.Mutex

	scanGroup := new(errgroup.Group)
	scanGroup.SetLimit(parallelism)
	for _, task := range allTasks {
		scanGroup.Go(func() error {
			(*task.resource).GetAndSetResourceConfig(configObj)

			collector.Emit(reporting.ScanProgress{
				ResourceType: (*task.resource).ResourceName(),
				Region:       task.region,
			})

			start := time.Now()
			identifiers, err := (*task.resource).GetAndSetIdentifiers(task.regionCtx, configObj)
			if err != nil {
				logging.Errorf("Unable to retrieve %v, %v", (*task.resource).ResourceName(), err)

				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:GetIdentifiers:%s", (*task.resource).ResourceName()),
				}, map[string]interface{}{
					"region": task.region,
				})

				collector.Emit(reporting.GeneralError{
					ResourceType: (*task.resource).ResourceName(),
					Description:  fmt.Sprintf("Unable to retrieve %s", (*task.resource).ResourceName()),
					Error:        err.Error(),
				})
				return nil
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("Done getting %s identifiers", (*task.resource).ResourceName()),
			}, map[string]interface{}{
				"recordCount": len(identifiers),
				"actionTime":  time.Since(start).Seconds(),
			})

			if len(identifiers) > 0 {
				logging.Infof("Found %d %s resources in %s", len(identifiers), (*task.resource).ResourceName(), task.region)

				foundMu.Lock()
				foundByRegion[task.region] = append(foundByRegion[task.region], indexedResource{task.idx, task.resource})
				foundMu.Unlock()

				for _, id := range identifiers {
					nukable, reason := true, ""
					if _, err := (*task.resource).IsNukable(id); err != nil {
						nukable, reason = false, err.Error()
					}
					collector.Emit(reporting.ResourceFound{
						ResourceType: (*task.resource).ResourceName(),
						Region:       task.region,
						Identifier:   id,
						Nukable:      nukable,
						Reason:       reason,
					})
				}
			}
			return nil
		})
	}
	if err := scanGroup.Wait(); err != nil {
		return nil, err
	}

	// Sort resources within each region by original registry index to preserve
	// the dependency ordering required for safe nuking (e.g. EC2 before VPCs).
	for region, found := range foundByRegion {
		slices.SortFunc(found, func(a, b indexedResource) int {
			return cmp.Compare(a.idx, b.idx)
		})
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

	slices.Sort(resourceTypes)
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
func NukeAllResources(ctx context.Context, account *AwsAccountResources, regions []string, parallelism int, collector *reporting.Collector) error {
	// Inject parallelism into context so batch_deleter (called via Nuke) can read it.
	ctx = context.WithValue(ctx, util.ParallelismKey, parallelism)
	p := util.GetParallelism(ctx)

	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	var mu sync.Mutex
	var allErrors *multierror.Error

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

	// Phase 1: nuke all regional resources in parallel. GlobalRegion is
	// intentionally excluded here because global resources (IAM, S3, CloudFront,
	// Route53) must be torn down after regional ones to avoid breaking in-flight
	// regional deletions that still depend on them.
	eg := new(errgroup.Group)
	eg.SetLimit(p)
	for _, region := range regions {
		if region == GlobalRegion {
			continue
		}
		eg.Go(func() error {
			nukeRegion(region)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		allErrors = multierror.Append(allErrors, err)
	}

	// Phase 2: nuke global resources after all regional nukes have completed.
	for _, region := range regions {
		if region == GlobalRegion {
			nukeRegion(region)
		}
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}
