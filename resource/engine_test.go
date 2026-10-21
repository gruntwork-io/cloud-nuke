package resource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClient stands in for a cloud SDK client. The fake provider below is the
// whole of a third cloud as the engine sees it: two subscriptions, one resource
// type, and the three judgements only a provider can make.
type fakeClient struct {
	subscription string

	// listErr, when set, is what listing answers instead of identifiers.
	listErr error

	mu      sync.Mutex
	deleted []string
}

// apiDisabledError stands in for the error a cloud answers when a service is
// switched off for this account.
type apiDisabledError struct{}

func (apiDisabledError) Error() string { return "service disabled for this subscription" }

// fakeProvider is a cloud with two subscriptions, the second of which holds the
// resources the first one's deletions depend on, so it is nuked last.
type fakeProvider struct {
	clients map[string]*fakeClient

	// beforeDelete runs before each delete is recorded, so a test can make the
	// order it asserts deterministic instead of hoping for one.
	beforeDelete func(target string)

	// throttleOnce makes the first delete of each target answer with a rate
	// limit, which IsThrottle then recognises.
	throttleOnce bool

	// configSpy records the resource config each lister was handed.
	configSpy func(config.ResourceType)

	// batchSize overrides the resource's batch size, so a test can ask for
	// more than one batch.
	batchSize int

	// nukeOrder records which target finished nuking, in order.
	mu        sync.Mutex
	nukeOrder []string
	throttled map[string]bool
}

func (p *fakeProvider) Targets() ([]Target, error) {
	return []Target{
		{ID: "sub-a", Label: "sub-a"},
		{ID: "sub-b", Label: "sub-b", Last: true},
	}, nil
}

func (p *fakeProvider) Prepare(ctx context.Context, target Target) (context.Context, []NukeableResource, error) {
	client := p.clients[target.ID]
	batchSize := p.batchSize
	if batchSize == 0 {
		batchSize = 10
	}

	r := &Resource[*fakeClient]{
		ResourceTypeName: "fake-widget",
		BatchSize:        batchSize,
		InitClient: func(r *Resource[*fakeClient], cfg any) {
			r.Client = client
			r.Scope = Scope{Region: target.ID}
		},
		ConfigGetter: func(c config.Config) config.ResourceType { return c.GCSBucket },
		Lister: func(ctx context.Context, c *fakeClient, scope Scope, cfg config.ResourceType) ([]*string, error) {
			if p.configSpy != nil {
				p.configSpy(cfg)
			}
			if c.listErr != nil {
				return nil, c.listErr
			}
			one := c.subscription + "-widget-1"
			two := c.subscription + "-widget-2"
			return []*string{&one, &two}, nil
		},
		Nuker: SequentialDeleter(func(ctx context.Context, c *fakeClient, id *string) error {
			if p.beforeDelete != nil {
				p.beforeDelete(target.ID)
			}

			p.mu.Lock()
			if p.throttleOnce && !p.throttled[target.ID] {
				p.throttled[target.ID] = true
				p.mu.Unlock()
				return rateLimitError{}
			}
			p.nukeOrder = append(p.nukeOrder, target.ID)
			p.mu.Unlock()

			c.mu.Lock()
			defer c.mu.Unlock()
			c.deleted = append(c.deleted, *id)
			return nil
		}),
	}
	r.Init(nil)
	return ctx, []NukeableResource{r}, nil
}

func (p *fakeProvider) SkipListError(err error, resourceType string, explicitlyRequested bool) bool {
	// errors.As rather than a type assertion, because the resource wraps what
	// its lister answered: a provider reading only the outermost error sees a
	// routine disabled service as a failure.
	var disabled apiDisabledError
	return errors.As(err, &disabled) && !explicitlyRequested
}

func (p *fakeProvider) IsThrottle(err error) bool {
	var throttled rateLimitError
	return errors.As(err, &throttled)
}

// rateLimitError stands in for the error a cloud answers when the caller is
// making requests too quickly.
type rateLimitError struct{}

func (rateLimitError) Error() string { return "rate limit exceeded, slow down" }

var _ Provider = (*fakeProvider)(nil)

// newFakeProvider builds a provider whose two subscriptions both hold widgets.
func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		clients: map[string]*fakeClient{
			"sub-a": {subscription: "sub-a"},
			"sub-b": {subscription: "sub-b"},
		},
		throttled: map[string]bool{},
	}
}

// recorder captures what the engine reported, which is what the user sees.
type recorder struct {
	mu       sync.Mutex
	found    []reporting.ResourceFound
	errs     []reporting.GeneralError
	gone     []reporting.ResourceDeleted
	progress []reporting.NukeProgress
}

func (r *recorder) OnEvent(event reporting.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch e := event.(type) {
	case reporting.ResourceFound:
		r.found = append(r.found, e)
	case reporting.GeneralError:
		r.errs = append(r.errs, e)
	case reporting.ResourceDeleted:
		r.gone = append(r.gone, e)
	case reporting.NukeProgress:
		r.progress = append(r.progress, e)
	}
}

func (r *recorder) collector() *reporting.Collector {
	collector := reporting.NewCollector()
	collector.AddRenderer(r)
	return collector
}

func TestScanWalksEveryTargetAProviderOffers(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{
		Include: func(string) bool { return true },
	}, rec.collector())
	require.NoError(t, err)

	// Two subscriptions, two widgets each, reported under their own labels.
	assert.Equal(t, 4, found.TotalResourceCount())
	assert.Len(t, found.Resources["sub-a"], 1)
	assert.Len(t, found.Resources["sub-b"], 1)

	labels := map[string]int{}
	for _, f := range rec.found {
		labels[f.Region]++
	}
	assert.Equal(t, map[string]int{"sub-a": 2, "sub-b": 2}, labels)
}

func TestNukeDefersTheTargetAProviderMarkedLast(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()

	// sub-a's deleter holds the lock until it is released, so a sub-b delete
	// that starts concurrently is recorded before sub-a finishes. Without this
	// the fake deleters do no work and the order comes out right by accident,
	// which let the deferral be deleted while this test still passed.
	release := make(chan struct{})
	p.beforeDelete = func(target string) {
		if target == "sub-a" {
			<-release
		}
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(release)
	}()

	rec := &recorder{}
	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	require.NoError(t, NukeAll(context.Background(), p, found, NukeRequest{
		Parallelism:   10,
		BatchPause:    time.Nanosecond,
		ThrottlePause: time.Nanosecond,
	}, rec.collector()))

	// Everything in sub-a is deleted before anything in sub-b, because sub-b
	// was marked Last: this is what AWS needs for its global resources.
	require.Len(t, p.nukeOrder, 4)
	assert.Equal(t, []string{"sub-a", "sub-a", "sub-b", "sub-b"}, p.nukeOrder)
	assert.Len(t, rec.gone, 4)
}

func TestScanSkipsADisabledServiceUnlessItWasAskedForByName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		explicit   bool
		wantErrors int
	}{
		{name: "not asked for by name, so stay quiet", explicit: false, wantErrors: 0},
		{name: "asked for by name, so say why", explicit: true, wantErrors: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := newFakeProvider()
			p.clients["sub-a"].listErr = apiDisabledError{}
			p.clients["sub-b"].listErr = apiDisabledError{}
			rec := &recorder{}

			found, err := Scan(context.Background(), p, ScanRequest{
				Include:             func(string) bool { return true },
				ExplicitlyRequested: func(string) bool { return tc.explicit },
			}, rec.collector())
			require.NoError(t, err)

			assert.Equal(t, 0, found.TotalResourceCount())
			assert.Len(t, rec.errs, tc.wantErrors*2, "one error per target")
		})
	}
}

func TestScanReportsAListFailureThatIsNotSkippable(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()
	p.clients["sub-a"].listErr = fmt.Errorf("the cloud is on fire")
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	// The failure is reported and the other target is still scanned.
	require.Len(t, rec.errs, 1)
	assert.Contains(t, rec.errs[0].Error, "the cloud is on fire")
	assert.Equal(t, 2, found.TotalResourceCount())
}

// sharedLabelProvider gives two distinct targets the same display label, which
// the contract says is cosmetic.
type sharedLabelProvider struct{ *fakeProvider }

func (p *sharedLabelProvider) Targets() ([]Target, error) {
	return []Target{
		{ID: "sub-a", Label: "my-tenant"},
		{ID: "sub-b", Label: "my-tenant"},
	}, nil
}

func TestTwoTargetsSharingALabelAreNotMerged(t *testing.T) {
	t.Parallel()

	p := &sharedLabelProvider{fakeProvider: newFakeProvider()}
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)
	assert.Equal(t, 4, found.TotalResourceCount(), "two subscriptions, two widgets each")

	require.NoError(t, NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector()))

	// Every widget is deleted exactly once.
	deleted := map[string]int{}
	for _, id := range append(p.clients["sub-a"].deleted, p.clients["sub-b"].deleted...) {
		deleted[id]++
	}
	assert.Equal(t, map[string]int{
		"sub-a-widget-1": 1, "sub-a-widget-2": 1,
		"sub-b-widget-1": 1, "sub-b-widget-2": 1,
	}, deleted)
}

// dryRunSuccessProvider has a resource whose permission verifier answers a
// successful check the way the EC2 dry run does, with a non-nil error.
type dryRunSuccessProvider struct{ deleted []string }

func (p *dryRunSuccessProvider) Targets() ([]Target, error) {
	return []Target{{ID: "us-east-1"}}, nil
}

func (p *dryRunSuccessProvider) Prepare(ctx context.Context, t Target) (context.Context, []NukeableResource, error) {
	r := &Resource[*fakeClient]{
		ResourceTypeName: "fake-volume",
		BatchSize:        10,
		InitClient: func(r *Resource[*fakeClient], cfg any) {
			r.Client = &fakeClient{subscription: t.ID}
			r.Scope = Scope{Region: t.ID}
		},
		ConfigGetter: func(c config.Config) config.ResourceType { return config.ResourceType{} },
		Lister: func(ctx context.Context, c *fakeClient, scope Scope, cfg config.ResourceType) ([]*string, error) {
			id := "vol-1"
			return []*string{&id}, nil
		},
		// A successful EC2 dry run returns this error, and most AWS verifiers
		// return it unchanged rather than mapping it to nil.
		PermissionVerifier: func(ctx context.Context, c *fakeClient, id *string) error {
			return errors.New("api error DryRunOperation: Request would have succeeded, but DryRun flag is set.")
		},
		Nuker: SequentialDeleter(func(ctx context.Context, c *fakeClient, id *string) error {
			p.deleted = append(p.deleted, *id)
			return nil
		}),
	}
	r.Init(nil)
	return ctx, []NukeableResource{r}, nil
}

func (p *dryRunSuccessProvider) SkipListError(error, string, bool) bool { return false }
func (p *dryRunSuccessProvider) IsThrottle(error) bool                  { return false }

func TestNukeStillDeletesWhatAVerifierCouldNotConfirm(t *testing.T) {
	t.Parallel()

	p := &dryRunSuccessProvider{}
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)
	require.Equal(t, 1, found.TotalResourceCount())

	// The scan reports it as not nukable, because that is what the verifier said.
	require.Len(t, rec.found, 1)
	assert.False(t, rec.found[0].Nukable)

	require.NoError(t, NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector()))

	// It is deleted anyway, which is what cloud-nuke has always done.
	assert.Equal(t, []string{"vol-1"}, p.deleted)
}

// orderedProvider registers three resource types whose listers answer at
// different speeds, so the order the scan finds them in is not registry order.
type orderedProvider struct {
	mu          sync.Mutex
	deleteOrder []string
	nukeErr     map[string]error
}

func (p *orderedProvider) Targets() ([]Target, error) {
	return []Target{{ID: "us-east-1"}}, nil
}

func (p *orderedProvider) Prepare(ctx context.Context, t Target) (context.Context, []NukeableResource, error) {
	// Registry order is the dependency order: instances before subnets before
	// the network, the way the AWS and GCP registries are written.
	names := []string{"instance", "subnet", "network"}
	delays := map[string]time.Duration{
		"instance": 30 * time.Millisecond, // answers last
		"subnet":   15 * time.Millisecond,
		"network":  0, // answers first
	}

	out := make([]NukeableResource, 0, len(names))
	for _, name := range names {
		r := &Resource[*fakeClient]{
			ResourceTypeName: name,
			BatchSize:        10,
			InitClient: func(r *Resource[*fakeClient], cfg any) {
				r.Client = &fakeClient{subscription: t.ID}
				r.Scope = Scope{Region: t.ID}
			},
			ConfigGetter: func(c config.Config) config.ResourceType { return config.ResourceType{} },
			Lister: func(ctx context.Context, c *fakeClient, scope Scope, cfg config.ResourceType) ([]*string, error) {
				time.Sleep(delays[name])
				id := name + "-1"
				return []*string{&id}, nil
			},
			Nuker: SequentialDeleter(func(ctx context.Context, c *fakeClient, id *string) error {
				p.mu.Lock()
				p.deleteOrder = append(p.deleteOrder, *id)
				p.mu.Unlock()
				return p.nukeErr[name]
			}),
		}
		r.Init(nil)
		out = append(out, r)
	}
	return ctx, out, nil
}

func (p *orderedProvider) SkipListError(error, string, bool) bool { return false }
func (p *orderedProvider) IsThrottle(error) bool                  { return false }

func TestNukeFollowsRegistryOrderWhateverOrderTheScanFinishedIn(t *testing.T) {
	t.Parallel()

	p := &orderedProvider{}
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	require.NoError(t, NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector()))

	// The network listed first and must still be deleted last, because the
	// subnets and instances in it have to go first.
	assert.Equal(t, []string{"instance-1", "subnet-1", "network-1"}, p.deleteOrder)
}

func TestNukeAggregatesEveryResourceTypesFailure(t *testing.T) {
	t.Parallel()

	p := &orderedProvider{nukeErr: map[string]error{
		"subnet":  errors.New("subnet has a dependent"),
		"network": errors.New("network still has a subnet"),
	}}
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	err = NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector())

	// Both failures survive, named by target and resource type, and the
	// instance that succeeded is not reported as a failure.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[us-east-1] subnet:")
	assert.Contains(t, err.Error(), "subnet-1: subnet has a dependent")
	assert.Contains(t, err.Error(), "[us-east-1] network:")
	assert.Contains(t, err.Error(), "network-1: network still has a subnet")
	assert.NotContains(t, err.Error(), "] instance:")

	// And the report says which identifiers failed and which did not.
	failed := map[string]bool{}
	for _, d := range rec.gone {
		failed[d.Identifier] = !d.Success
	}
	assert.Equal(t, map[string]bool{"instance-1": false, "subnet-1": true, "network-1": true}, failed)
}

// failingPrepareProvider cannot build a client for its only target.
type failingPrepareProvider struct{}

func (failingPrepareProvider) Targets() ([]Target, error) {
	return []Target{{ID: "us-east-1"}}, nil
}

func (failingPrepareProvider) Prepare(context.Context, Target) (context.Context, []NukeableResource, error) {
	return nil, nil, errors.New("no credentials for us-east-1")
}

func (failingPrepareProvider) SkipListError(error, string, bool) bool { return false }
func (failingPrepareProvider) IsThrottle(error) bool                  { return false }

func TestScanFailsLoudlyWhenAProviderCannotPrepareATarget(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	_, err := Scan(context.Background(), failingPrepareProvider{}, ScanRequest{
		Include: func(string) bool { return true },
	}, rec.collector())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no credentials for us-east-1")
}

// failingTargetsProvider cannot work out what to scan.
type failingTargetsProvider struct{ failingPrepareProvider }

func (failingTargetsProvider) Targets() ([]Target, error) {
	return nil, errors.New("could not list regions")
}

func TestScanFailsLoudlyWhenAProviderCannotListItsTargets(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	_, err := Scan(context.Background(), failingTargetsProvider{}, ScanRequest{
		Include: func(string) bool { return true },
	}, rec.collector())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not list regions")
}

// duplicateTargetProvider returns the same target twice, which a provider that
// forgets to de-duplicate its region flag would do.
type duplicateTargetProvider struct{ *fakeProvider }

func (p *duplicateTargetProvider) Targets() ([]Target, error) {
	return []Target{{ID: "sub-a"}, {ID: "sub-a"}}, nil
}

func TestScanRefusesTheSameTargetTwice(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	_, err := Scan(context.Background(), &duplicateTargetProvider{fakeProvider: newFakeProvider()},
		ScanRequest{Include: func(string) bool { return true }}, rec.collector())

	// Scanning it twice would delete everything in it twice, so the run stops
	// and the error names the target the caller repeated.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sub-a")
}

func TestNukeRefusesToStrandFindingsOutsideItsTargets(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	// A caller that nukes a different set of targets than it scanned, which is
	// what a multi-project GCP run would do by accident, is told rather than
	// quietly leaving those resources alive.
	found.Targets = []Target{{ID: "sub-a"}}

	err = NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `found resources in "sub-b"`)
}

// TestScanObeysTheResourceTypeFilterAndNukeDeletesOnlyWhatItKept pins that a
// resource type the user excluded is never listed, never reported and never
// deleted. Nothing between the CLI flag and the delete call pinned this: the
// filter is one call to req.Include, and removing it left the suite green.
func TestScanObeysTheResourceTypeFilterAndNukeDeletesOnlyWhatItKept(t *testing.T) {
	t.Parallel()

	p := &orderedProvider{}
	rec := &recorder{}

	// The user asked for everything except the network.
	found, err := Scan(context.Background(), p, ScanRequest{
		Include: func(resourceType string) bool { return resourceType != "network" },
	}, rec.collector())
	require.NoError(t, err)

	assert.Equal(t, 2, found.TotalResourceCount())
	var reported []string
	for _, f := range rec.found {
		reported = append(reported, f.ResourceType)
	}
	assert.ElementsMatch(t, []string{"instance", "subnet"}, reported)

	require.NoError(t, NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector()))

	assert.Equal(t, []string{"instance-1", "subnet-1"}, p.deleteOrder)
}

// TestScanGivesEachResourceTheCallersConfig pins that the config file reaches
// the listers. Passing an empty config instead silently stopped every
// name, time and tag filter from applying, and no test noticed.
func TestScanGivesEachResourceTheCallersConfig(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()
	var seen []config.ResourceType
	var seenMu sync.Mutex
	p.configSpy = func(cfg config.ResourceType) {
		seenMu.Lock()
		defer seenMu.Unlock()
		seen = append(seen, cfg)
	}

	// A config the caller set, which the lister must receive.
	wanted := config.Config{}
	wanted.GCSBucket.IncludeRule.NamesRegExp = []config.Expression{{RE: *regexp.MustCompile("^keep-")}}

	rec := &recorder{}
	_, err := Scan(context.Background(), p, ScanRequest{
		Include: func(string) bool { return true },
		Config:  wanted,
	}, rec.collector())
	require.NoError(t, err)

	require.Len(t, seen, 2, "one per target")
	for _, cfg := range seen {
		require.Len(t, cfg.IncludeRule.NamesRegExp, 1)
		assert.Equal(t, "^keep-", cfg.IncludeRule.NamesRegExp[0].RE.String())
	}
}

func TestNukeWaitsOutAThrottleAndStillDeletesTheRest(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()
	p.throttleOnce = true
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	err = NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector())

	// A throttle is waited out rather than recorded as a failure, so the run
	// reports no error. Both subscriptions lose their second widget; the first
	// is the one the rate limit swallowed, which is a known gap in the engine
	// it inherited and the reason this test exists to describe it.
	require.NoError(t, err)
	assert.Equal(t, []string{"sub-a-widget-2"}, p.clients["sub-a"].deleted)
	assert.Equal(t, []string{"sub-b-widget-2"}, p.clients["sub-b"].deleted)
}

func TestNukeSplitsIntoBatchesOfTheResourcesMaxSize(t *testing.T) {
	t.Parallel()

	p := newFakeProvider()
	p.batchSize = 1
	rec := &recorder{}

	found, err := Scan(context.Background(), p, ScanRequest{Include: func(string) bool { return true }}, rec.collector())
	require.NoError(t, err)

	require.NoError(t, NukeAll(context.Background(), p, found, NukeRequest{
		BatchPause: time.Nanosecond, ThrottlePause: time.Nanosecond,
	}, rec.collector()))

	// Two identifiers per target at a batch size of one is two batches each,
	// which the progress events report so a renderer can show them.
	batches := 0
	for _, event := range rec.progress {
		batches++
		assert.Equal(t, 1, event.BatchSize)
	}
	assert.Equal(t, 4, batches)
}
