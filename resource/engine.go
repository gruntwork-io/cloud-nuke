package resource

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// Target is one unit of work the engine walks: an AWS region, a GCP project, an
// Azure subscription. The engine never interprets a target beyond ordering and
// labeling it; what a target means belongs to the Provider that produced it.
type Target struct {
	// ID is the value the provider recognizes, such as "us-east-1" or a project
	// id. It is also this target's identity: the engine keys everything it holds
	// per target on it, so two targets in one run never share an ID.
	ID string

	// Label is what the user sees: the report's Region column and the "region"
	// key of the JSON output. It defaults to ID when empty. Two targets may
	// share a label, which is why the engine keys on ID rather than on this.
	Label string

	// Last defers this target until every other target has finished nuking.
	// AWS sets it on the global pseudo-region, whose resources (IAM, S3,
	// CloudFront, Route53) regional deletions still depend on.
	Last bool
}

// label returns what the user should see for this target.
func (t Target) label() string {
	if t.Label != "" {
		return t.Label
	}
	return t.ID
}

// Provider is what a cloud provider implements to be scanned and nuked. It is
// the whole provider-facing contract: everything else in a provider package is
// its own resources, its own clients and its own CLI wiring.
type Provider interface {
	// Targets returns the units of work to walk, in the order they should be
	// scanned. One per region for AWS, exactly one per project for GCP.
	Targets() ([]Target, error)

	// Prepare builds the clients for one target and returns the resources to
	// scan, in registry order, since that order is the dependency order the
	// nuke later relies on. The returned context is the one the engine passes
	// back to those resources, so a provider can attach per-target values such
	// as an account id.
	Prepare(ctx context.Context, target Target) (context.Context, []NukeableResource, error)

	// SkipListError reports whether a failure to list one resource type should
	// be passed over in silence rather than reported. GCP skips a disabled API;
	// AWS skips nothing. explicitlyRequested is true when the user named this
	// resource type on the command line, in which case silence would be wrong.
	SkipListError(err error, resourceType string, explicitlyRequested bool) bool

	// IsThrottle reports whether a nuke failure is the provider's rate limit,
	// which the engine answers by waiting rather than by recording a failure.
	IsThrottle(err error) bool
}

// Found holds what a scan discovered, keyed by Target.ID, with each target's
// resources left in registry order.
type Found struct {
	Targets   []Target
	Resources map[string][]NukeableResource
}

// TotalResourceCount returns how many individual resources were found.
func (f *Found) TotalResourceCount() int {
	total := 0
	for _, resources := range f.Resources {
		for _, r := range resources {
			total += len(r.ResourceIdentifiers())
		}
	}
	return total
}

// ScanRequest is what a scan needs beyond the provider itself.
type ScanRequest struct {
	// Include reports whether a resource type should be scanned at all.
	Include func(resourceType string) bool

	// ExplicitlyRequested reports whether the user named this resource type,
	// which decides whether a list failure may be skipped quietly.
	ExplicitlyRequested func(resourceType string) bool

	Config      config.Config
	Parallelism int
}

// Scan walks every target the provider offers and returns what it found. One
// errgroup covers every target and every resource type, so a parallelism of N
// means at most N simultaneous API calls in total rather than N per target.
func Scan(ctx context.Context, p Provider, req ScanRequest, collector *reporting.Collector) (*Found, error) {
	targets, err := p.Targets()
	if err != nil {
		return nil, err
	}

	if err := refuseDuplicateTargets(targets); err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, util.ParallelismKey, req.Parallelism)
	parallelism := util.GetParallelism(ctx)

	// Phase 1: build the clients for every target. A provider may call its
	// cloud here, as the AWS one does to read the account id, so this runs
	// under the same limit as the scan rather than all at once: without it a
	// sweep of thirty regions opens thirty calls before any limit applies, and
	// a rate limit there ends the whole sweep with nothing deleted.
	type prepared struct {
		ctx       context.Context
		resources []NukeableResource
	}
	prep := make(map[string]*prepared, len(targets))
	var prepMu sync.Mutex

	prepGroup := new(errgroup.Group)
	prepGroup.SetLimit(parallelism)
	for _, target := range targets {
		prepGroup.Go(func() error {
			targetCtx, resources, err := p.Prepare(ctx, target)
			if err != nil {
				return err
			}
			prepMu.Lock()
			prep[target.ID] = &prepared{ctx: targetCtx, resources: resources}
			prepMu.Unlock()
			return nil
		})
	}
	if err := prepGroup.Wait(); err != nil {
		return nil, err
	}

	// Phase 2: scan every included resource type on every target under one limit.
	type task struct {
		target   Target
		ctx      context.Context
		idx      int
		resource NukeableResource
	}
	var tasks []task
	for _, target := range targets {
		ready, ok := prep[target.ID]
		if !ok {
			continue
		}
		for i, r := range ready.resources {
			if req.Include != nil && !req.Include(r.ResourceName()) {
				continue
			}
			tasks = append(tasks, task{target: target, ctx: ready.ctx, idx: i, resource: r})
		}
	}

	type indexed struct {
		idx      int
		resource NukeableResource
	}
	foundByTarget := make(map[string][]indexed)
	var foundMu sync.Mutex

	scanGroup := new(errgroup.Group)
	scanGroup.SetLimit(parallelism)
	for _, t := range tasks {
		scanGroup.Go(func() error {
			// A cancelled run stops listing. Without this, every task already
			// queued still calls the cloud after the caller gave up.
			if err := t.ctx.Err(); err != nil {
				return err
			}

			resourceType := t.resource.ResourceName()
			t.resource.GetAndSetResourceConfig(req.Config)

			collector.Emit(reporting.ScanProgress{
				ResourceType: resourceType,
				Region:       t.target.label(),
			})

			start := time.Now()
			identifiers, err := t.resource.GetAndSetIdentifiers(t.ctx, req.Config)
			if err != nil {
				explicit := req.ExplicitlyRequested != nil && req.ExplicitlyRequested(resourceType)
				if p.SkipListError(err, resourceType, explicit) {
					logging.Debugf("Skipping %s in %s: %v", resourceType, t.target.label(), err)
					return nil
				}
				logging.Errorf("Unable to retrieve %v, %v", resourceType, err)
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:GetIdentifiers:%s", resourceType),
				}, map[string]interface{}{"region": t.target.label()})
				collector.Emit(reporting.GeneralError{
					ResourceType: resourceType,
					Description:  fmt.Sprintf("Unable to retrieve %s", resourceType),
					Error:        err.Error(),
				})
				return nil
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("Done getting %s identifiers", resourceType),
			}, map[string]interface{}{
				"recordCount": len(identifiers),
				"actionTime":  time.Since(start).Seconds(),
			})

			if len(identifiers) == 0 {
				return nil
			}

			logging.Infof("Found %d %s resources in %s", len(identifiers), resourceType, t.target.label())

			foundMu.Lock()
			foundByTarget[t.target.ID] = append(foundByTarget[t.target.ID], indexed{t.idx, t.resource})
			foundMu.Unlock()

			for _, id := range identifiers {
				nukable, reason := true, ""
				if _, err := t.resource.IsNukable(id); err != nil {
					nukable, reason = false, err.Error()
				}
				collector.Emit(reporting.ResourceFound{
					ResourceType: resourceType,
					Region:       t.target.label(),
					Identifier:   id,
					Nukable:      nukable,
					Reason:       reason,
				})
			}
			return nil
		})
	}
	if err := scanGroup.Wait(); err != nil {
		return nil, err
	}

	// Restore registry order within each target, which is the order the nuke
	// relies on to delete a resource before whatever it depends on.
	found := &Found{Targets: targets, Resources: map[string][]NukeableResource{}}
	for id, discovered := range foundByTarget {
		slices.SortFunc(discovered, func(a, b indexed) int {
			return cmp.Compare(a.idx, b.idx)
		})
		ordered := make([]NukeableResource, len(discovered))
		for i, d := range discovered {
			ordered[i] = d.resource
		}
		found.Resources[id] = ordered
	}

	logging.Info("Done searching for resources")
	logging.Infof("Found total of %d resources", found.TotalResourceCount())

	return found, nil
}

// refuseDuplicateTargets fails a run whose targets share an ID. The engine keys
// everything it holds per target on that ID, so a repeat would scan one target
// twice and send each identifier to the delete twice.
func refuseDuplicateTargets(targets []Target) error {
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if _, duplicate := seen[target.ID]; duplicate {
			return fmt.Errorf("%q was given twice, which would scan and delete it twice", target.ID)
		}
		seen[target.ID] = struct{}{}
	}
	return nil
}

// pause waits, and answers the cancellation instead when the caller gives up
// first. A bare sleep here means a cancelled run burns a whole throttle pause,
// which is a minute by default, and then deletes the next batch anyway.
func pause(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DefaultBatchPause is how long the engine waits between batches of one
// resource type, and DefaultThrottlePause how long it waits out a rate limit.
const (
	DefaultBatchPause    = 10 * time.Second
	DefaultThrottlePause = 1 * time.Minute
)

// NukeRequest is what a nuke needs beyond the provider and what was found. The
// two pauses are settable so a test can run the engine without waiting them out.
type NukeRequest struct {
	Parallelism   int
	BatchPause    time.Duration
	ThrottlePause time.Duration
}

// withDefaults fills in the pauses a caller left at zero.
func (r NukeRequest) withDefaults() NukeRequest {
	if r.BatchPause == 0 {
		r.BatchPause = DefaultBatchPause
	}
	if r.ThrottlePause == 0 {
		r.ThrottlePause = DefaultThrottlePause
	}
	return r
}

// throttleShape reports whether a batch's failure holds a rate limit, and
// whether it holds nothing else. Resource.Nuke returns one error per failed
// identifier in a multierror, so asking the provider about the aggregate
// answers for whichever error it unwraps first.
func throttleShape(p Provider, err error) (rateLimited, onlyRateLimits bool) {
	failures := flatten(err)
	onlyRateLimits = len(failures) > 0
	for _, failure := range failures {
		if p.IsThrottle(failure) {
			rateLimited = true
		} else {
			onlyRateLimits = false
		}
	}
	return rateLimited, onlyRateLimits
}

// flatten returns the failures inside an aggregate, however deeply they are
// nested. One identifier's failure can be an aggregate of its own: an EKS
// cluster answers for its node groups and its Fargate profiles. Asking the
// provider about such a value answers for whichever error it unwraps first, so
// a rate limit beside a permanent failure would read as a rate limit, and the
// permanent failure would leave the run with the skipped batch.
func flatten(err error) []error {
	if err == nil {
		return nil
	}

	var merr *multierror.Error
	if errors.As(err, &merr) && len(merr.Errors) > 0 {
		var failures []error
		for _, e := range merr.Errors {
			failures = append(failures, flatten(e)...)
		}
		return failures
	}

	// What errors.Join builds, which a resource deleting several things for one
	// identifier may hand back.
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		if parts := joined.Unwrap(); len(parts) > 0 {
			var failures []error
			for _, e := range parts {
				failures = append(failures, flatten(e)...)
			}
			return failures
		}
	}

	return []error{err}
}

// NukeAll deletes everything a scan found. Targets run concurrently up to
// parallelism, except the ones a provider marked Last, which run after the rest.
//
// Findings under an ID that is not one of the targets are a hard error and
// nothing is deleted, since a caller who scanned one set of targets and nukes
// another would otherwise leave those resources alive and report success.
func NukeAll(ctx context.Context, p Provider, found *Found, req NukeRequest, collector *reporting.Collector) error {
	req = req.withDefaults()
	ctx = context.WithValue(ctx, util.ParallelismKey, req.Parallelism)
	limit := util.GetParallelism(ctx)

	// A nuke is reached directly by a library caller, so it checks what a scan
	// checks: two targets sharing an ID would delete one target's resources
	// twice over, from two goroutines holding the same resource.
	if err := refuseDuplicateTargets(found.Targets); err != nil {
		return err
	}

	// Everything the scan found belongs to a target the nuke walks. Without
	// this, a provider that reports findings under an ID it does not return
	// from Targets leaves them undeleted and says nothing.
	covered := make(map[string]struct{}, len(found.Targets))
	for _, target := range found.Targets {
		covered[target.ID] = struct{}{}
	}
	for id := range found.Resources {
		if _, ok := covered[id]; !ok {
			return fmt.Errorf("found resources in %q, which is not one of the targets to nuke", id)
		}
	}

	collector.Emit(reporting.NukeStarted{Total: found.TotalResourceCount()})
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	var mu sync.Mutex
	var allErrors *multierror.Error

	nukeTarget := func(target Target) {
		if err := nukeOneTarget(ctx, p, found, target, req, collector); err != nil {
			mu.Lock()
			allErrors = multierror.Append(allErrors, err)
			mu.Unlock()
		}

		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done nuking target",
		}, map[string]interface{}{
			"region":        target.label(),
			"resourceCount": len(found.Resources[target.ID]),
		})
	}

	eg := new(errgroup.Group)
	eg.SetLimit(limit)
	for _, target := range found.Targets {
		if target.Last {
			continue
		}
		eg.Go(func() error {
			nukeTarget(target)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		allErrors = multierror.Append(allErrors, err)
	}

	for _, target := range found.Targets {
		if target.Last {
			nukeTarget(target)
		}
	}

	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}

// nukeOneTarget deletes every resource type found in one target, in the order
// the scan left them.
func nukeOneTarget(ctx context.Context, p Provider, found *Found, target Target, req NukeRequest, collector *reporting.Collector) error {
	var allErrors *multierror.Error

	for _, r := range found.Resources[target.ID] {
		// A cancelled run stops deleting. Waking from a pause is not enough on
		// its own: the next batch would go out after the caller asked the run
		// to stop, and the run would report success.
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("[%s] %w", target.label(), err)
		}

		// Every identifier the scan found is sent to the provider, including one
		// IsNukable reported as not nukable. The GCP loop used to filter here and
		// no GCP resource sets a PermissionVerifier, so that filter never did
		// anything; the first GCP resource to add one will be deleted even though
		// the scan reported it as not nukable. A dry-run verifier answers a
		// successful check with the error "Request would have succeeded, but
		// DryRun flag is set", and 15 of the 22 AWS verifiers return that error
		// unchanged, so filtering on IsNukable here would stop deleting EBS
		// volumes, Elastic IPs, key pairs and every IPAM resource. The scan
		// reports what it could not verify; the delete still tries.
		identifiers := r.ResourceIdentifiers()
		if len(identifiers) == 0 {
			continue
		}

		batches := util.Split(identifiers, r.MaxBatchSize())
		for i, batch := range batches {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("[%s] %w", target.label(), err)
			}

			collector.Emit(reporting.NukeProgress{
				ResourceType: r.ResourceName(),
				Region:       target.label(),
				BatchSize:    len(batch),
			})

			results, err := r.Nuke(ctx, batch)

			for _, result := range results {
				errStr := ""
				if result.Error != nil {
					errStr = result.Error.Error()
				}
				collector.Emit(reporting.ResourceDeleted{
					ResourceType: r.ResourceName(),
					Region:       target.label(),
					Identifier:   result.Identifier,
					Success:      result.Error == nil,
					Warning:      result.Error != nil && util.IsWarningError(result.Error),
					Error:        errStr,
				})
			}

			if err != nil {
				// A batch fails one identifier at a time, so it can hold a rate
				// limit and a real failure together. The engine waits out the
				// limit either way, and only skips the batch when the limit is
				// all there was: otherwise a single throttled identifier would
				// carry every failure beside it out of the run's errors, and
				// the run would exit clean over resources it did not delete.
				rateLimited, onlyRateLimits := throttleShape(p, err)
				if rateLimited {
					logging.Debug("Request limit reached. Waiting before making new requests")
					if err := pause(ctx, req.ThrottlePause); err != nil {
						return fmt.Errorf("[%s] %w", target.label(), err)
					}
				}
				if onlyRateLimits {
					continue
				}

				allErrors = multierror.Append(allErrors,
					fmt.Errorf("[%s] %s: %w", target.label(), r.ResourceName(), err))

				// Aggregated failure counts per resource type, which is what the
				// per-provider loops reported before the engine owned this.
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:Nuke:%s", r.ResourceName()),
				}, map[string]interface{}{"region": target.label()})
			}

			if i != len(batches)-1 {
				logging.Debugf("Sleeping for %s before processing next batch...", req.BatchPause)
				if err := pause(ctx, req.BatchPause); err != nil {
					return fmt.Errorf("[%s] %w", target.label(), err)
				}
			}
		}
	}

	return allErrors.ErrorOrNil()
}
