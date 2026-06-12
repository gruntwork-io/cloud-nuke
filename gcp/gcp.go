package gcp

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// IsNukeable checks whether a resource type should be nuked based on the
// requested resource types and exclude lists. An empty include list or the
// special value "all" means nuke everything, minus any excluded types.
func IsNukeable(resourceType string, resourceTypes []string, excludeResourceTypes []string) bool {
	if collections.ListContainsElement(excludeResourceTypes, resourceType) {
		return false
	}
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}

// GetAllResources lists all GCP resources that can be deleted.
func GetAllResources(ctx context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*GcpProjectResources, error) {
	allResources := GcpProjectResources{
		Resources: map[string]GcpResources{},
	}

	ctx = context.WithValue(ctx, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)
	// Inject parallelism from the query so downstream callers can read it via
	// util.GetParallelism without needing the caller to set it on the context.
	ctx = context.WithValue(ctx, util.ParallelismKey, query.Parallelism)
	parallelism := util.GetParallelism(ctx)

	type indexedResource struct {
		idx int
		res *GcpResource
	}
	type regionSetup struct {
		nukeable []indexedResource
	}

	// Phase 1: init resources for each region concurrently.
	// GetAndInitRegisteredResources only allocates GCP clients (no API calls), so
	// no concurrency limit is needed here.
	setups := make(map[string]*regionSetup, len(query.Regions))
	var setupMu sync.Mutex

	setupGroup := new(errgroup.Group)
	for _, region := range query.Regions {
		setupGroup.Go(func() error {
			cfg := resources.GcpConfig{ProjectID: query.ProjectID, Region: region}
			regionResources := GetAndInitRegisteredResources(cfg, region)
			setup := &regionSetup{}
			for i, res := range regionResources {
				resourceName := (*res).ResourceName()
				if IsNukeable(resourceName, query.ResourceTypes, query.ExcludeResourceTypes) {
					setup.nukeable = append(setup.nukeable, indexedResource{idx: i, res: res})
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
		region string
		idx    int
		res    *GcpResource
	}
	var allTasks []resourceTask
	for _, region := range query.Regions {
		setup, ok := setups[region]
		if !ok {
			continue
		}
		for _, r := range setup.nukeable {
			allTasks = append(allTasks, resourceTask{region: region, idx: r.idx, res: r.res})
		}
	}

	foundByRegion := make(map[string][]indexedResource)
	var foundMu sync.Mutex

	scanGroup := new(errgroup.Group)
	scanGroup.SetLimit(parallelism)
	for _, task := range allTasks {
		scanGroup.Go(func() error {
			resourceName := (*task.res).ResourceName()
			collector.Emit(reporting.ScanProgress{
				ResourceType: resourceName,
				Region:       task.region,
			})

			identifiers, err := (*task.res).GetAndSetIdentifiers(ctx, configObj)
			if err != nil {
				if isServiceDisabledError(err) && !collections.ListContainsElement(query.ResourceTypes, resourceName) {
					logging.Debugf("Skipping %s: API is disabled in this project", resourceName)
					return nil
				}
				logging.Debugf("Error getting identifiers for %s: %v", resourceName, err)
				collector.Emit(reporting.GeneralError{
					ResourceType: resourceName,
					Description:  fmt.Sprintf("Unable to retrieve %s", resourceName),
					Error:        err.Error(),
				})
				return nil
			}

			if len(identifiers) > 0 {
				logging.Infof("Found %d %s resources", len(identifiers), resourceName)

				foundMu.Lock()
				foundByRegion[task.region] = append(foundByRegion[task.region], indexedResource{task.idx, task.res})
				foundMu.Unlock()

				for _, id := range identifiers {
					nukable, reason := true, ""
					if _, err := (*task.res).IsNukable(id); err != nil {
						nukable, reason = false, err.Error()
					}
					collector.Emit(reporting.ResourceFound{
						ResourceType: resourceName,
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
	// the dependency ordering required for safe nuking.
	for region, found := range foundByRegion {
		slices.SortFunc(found, func(a, b indexedResource) int {
			return cmp.Compare(a.idx, b.idx)
		})
		regionList := make([]*GcpResource, len(found))
		for i, r := range found {
			regionList[i] = r.res
		}
		allResources.Resources[region] = GcpResources{Resources: regionList}
	}

	logging.Info("Done searching for GCP resources")
	logging.Infof("Found total of %d GCP resources", allResources.TotalResourceCount())

	return &allResources, nil
}

// NukeAllResources nukes all GCP resources across the given regions.
func NukeAllResources(ctx context.Context, account *GcpProjectResources, regions []string, parallelism int, collector *reporting.Collector) error {
	// Inject parallelism into context so batch_deleter (called via Nuke) can read it.
	ctx = context.WithValue(ctx, util.ParallelismKey, parallelism)
	p := util.GetParallelism(ctx)

	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})

	eg := new(errgroup.Group)
	eg.SetLimit(p)
	var mu sync.Mutex
	var allErrors *multierror.Error

	for _, region := range regions {
		eg.Go(func() error {
			if err := nukeAllResourcesInRegion(ctx, account, region, collector); err != nil {
				mu.Lock()
				allErrors = multierror.Append(allErrors, err)
				mu.Unlock()
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		allErrors = multierror.Append(allErrors, err)
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}

// nukeAllResourcesInRegion nukes all resources in a single region.
func nukeAllResourcesInRegion(ctx context.Context, account *GcpProjectResources, region string, collector *reporting.Collector) error {
	var allErrors *multierror.Error

	resourcesInRegion := account.Resources[region]
	for _, gcpResource := range resourcesInRegion.Resources {
		if err := nukeResource(ctx, gcpResource, region, collector); err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
	}

	return allErrors.ErrorOrNil()
}

// nukeResource nukes a single GCP resource type
func nukeResource(ctx context.Context, gcpResource *GcpResource, region string, collector *reporting.Collector) error {
	// Filter to only nukable resources
	var nukableIdentifiers []string
	for _, id := range (*gcpResource).ResourceIdentifiers() {
		if nukable, reason := (*gcpResource).IsNukable(id); !nukable {
			logging.Debugf("[Skipping] %s %s because %v", (*gcpResource).ResourceName(), id, reason)
			continue
		}
		nukableIdentifiers = append(nukableIdentifiers, id)
	}

	if len(nukableIdentifiers) == 0 {
		return nil
	}

	// Split API calls into batches
	logging.Debugf("Terminating %d %s in batches", len(nukableIdentifiers), (*gcpResource).ResourceName())
	batches := util.Split(nukableIdentifiers, (*gcpResource).MaxBatchSize())

	var allErrors *multierror.Error

	for i, batch := range batches {
		// Emit progress event (CLIRenderer updates its progress bar)
		collector.Emit(reporting.NukeProgress{
			ResourceType: (*gcpResource).ResourceName(),
			Region:       region,
			BatchSize:    len(batch),
		})

		results, err := (*gcpResource).Nuke(ctx, batch)

		// Emit ResourceDeleted for each result
		for _, result := range results {
			errStr := ""
			if result.Error != nil {
				errStr = result.Error.Error()
			}
			collector.Emit(reporting.ResourceDeleted{
				ResourceType: (*gcpResource).ResourceName(),
				Region:       region,
				Identifier:   result.Identifier,
				Success:      result.Error == nil,
				Error:        errStr,
			})
		}

		if err != nil {
			if isQuotaExhaustedError(err) {
				logging.Debug(
					"Quota exceeded. Waiting 1 minute before making new requests",
				)
				time.Sleep(1 * time.Minute)
				continue
			}

			allErrors = multierror.Append(allErrors, fmt.Errorf("[%s] %s: %w", region, (*gcpResource).ResourceName(), err))

			// Report to telemetry - aggregated metrics of failures per resources.
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("error:Nuke:%s", (*gcpResource).ResourceName()),
			}, map[string]interface{}{
				"region": region,
			})
		}

		if i != len(batches)-1 {
			logging.Debug("Sleeping for 10 seconds before processing next batch...")
			time.Sleep(10 * time.Second)
		}
	}

	return allErrors.ErrorOrNil()
}

// ListResourceTypes returns a sorted list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	var resourceTypes []string
	for _, r := range GetAllRegisteredResources() {
		resourceTypes = append(resourceTypes, (*r).ResourceName())
	}
	slices.Sort(resourceTypes)
	return resourceTypes
}
