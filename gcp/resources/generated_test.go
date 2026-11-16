package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/james00012/cloud-inventory/generic"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// descriptor reads what the inventory generator produced for GCS buckets. No
// part of it was written by hand, which is the claim this file is testing.
func descriptor(t *testing.T) generic.Descriptor {
	t.Helper()

	body, err := os.ReadFile("testdata/gcs-bucket.descriptor.json")
	require.NoError(t, err)

	var doc struct {
		Descriptors map[string]generic.Descriptor `json:"descriptors"`
	}
	require.NoError(t, json.Unmarshal(body, &doc))

	found, ok := doc.Descriptors["buckets"]
	require.True(t, ok, "the generator produced no descriptor for buckets")
	return found
}

// bucketAPI stands in for Cloud Storage. refuse makes a listing fail the way a
// project the caller cannot fully read does.
type bucketAPI struct {
	refuse  bool
	deleted []string
	buckets []map[string]any
}

func (b *bucketAPI) serve(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodDelete {
			b.deleted = append(b.deleted, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if b.refuse {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "caller lacks storage.buckets.list"},
			})
			return
		}
		for i := range b.buckets {
			b.buckets[i]["selfLink"] = "http://" + r.Host + "/b/" + b.buckets[i]["name"].(string)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": b.buckets})
	}))
	t.Cleanup(server.Close)
	return server
}

func bucket(name string, age time.Duration) map[string]any {
	return map[string]any{
		"name":        name,
		"timeCreated": time.Now().Add(-age).UTC().Format(time.RFC3339),
		"labels":      map[string]any{"owner": "team"},
	}
}

func generatedBuckets(t *testing.T, api *bucketAPI, cfg config.ResourceType) GcpResource {
	t.Helper()

	server := api.serve(t)
	return NewGeneratedResource(Generated{
		TypeName:      "gcs-bucket",
		Descriptor:    descriptor(t),
		BaseURL:       server.URL,
		NewHTTPClient: func(context.Context) (*http.Client, error) { return server.Client(), nil },
		ConfigGetter:  func(config.Config) config.ResourceType { return cfg },
		ScopeOf: func(gcpCfg GcpConfig) generic.Scope {
			return generic.Scope{"project": {gcpCfg.ProjectID}}
		},
		LocationScope: LocationAppliedByLister,
	})
}

func listThrough(t *testing.T, r GcpResource) []string {
	t.Helper()

	r.Init(GcpConfig{ProjectID: "demo"})
	found, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)
	return found
}

// TestAGeneratedResourceListsAndNukes drives a resource type nobody wrote code
// for through cloud-nuke's own machinery.
func TestAGeneratedResourceListsAndNukes(t *testing.T) {
	api := &bucketAPI{buckets: []map[string]any{
		bucket("demo-assets", 48*time.Hour),
		bucket("demo-logs", time.Hour),
	}}
	r := generatedBuckets(t, api, config.ResourceType{})

	found := listThrough(t, r)
	require.Len(t, found, 2)

	results, err := r.Nuke(context.Background(), found)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Len(t, api.deleted, 2, "both buckets should have been deleted")
	assert.Contains(t, strings.Join(api.deleted, " "), "/b/demo-assets")
}

// TestCloudNukesOwnFiltersStillDecide pins the split: the library reads the
// project, and which of those buckets to act on stays here.
func TestCloudNukesOwnFiltersStillDecide(t *testing.T) {
	api := &bucketAPI{buckets: []map[string]any{
		bucket("demo-assets", 48*time.Hour),
		bucket("keep-me", 48*time.Hour),
		bucket("demo-logs", time.Minute),
	}}

	olderThan := time.Now().Add(-time.Hour)
	r := generatedBuckets(t, api, config.ResourceType{
		IncludeRule: config.FilterRule{NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^demo-")}}},
		ExcludeRule: config.FilterRule{TimeAfter: &olderThan},
	})

	found := listThrough(t, r)

	// demo-assets alone: keep-me fails the name rule and demo-logs is too new.
	require.Len(t, found, 1)
	assert.Contains(t, found[0], "demo-assets")
}

// TestAProjectItCouldNotFullyReadWarnsAndCarriesOn is the policy that differs
// from a test suite's. cloud-nuke reports the gap and nukes what it could
// read; terratest takes the same report and fails the test.
func TestAProjectItCouldNotFullyReadWarnsAndCarriesOn(t *testing.T) {
	// Warnings go through pterm rather than the logrus logger, so capturing
	// the logger would quietly assert on nothing.
	var logged bytes.Buffer
	previous := pterm.Warning.Writer
	pterm.Warning.Writer = &logged
	t.Cleanup(func() { pterm.Warning.Writer = previous })

	api := &bucketAPI{refuse: true}
	r := generatedBuckets(t, api, config.ResourceType{})

	r.Init(GcpConfig{ProjectID: "demo"})
	found, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})

	// The run carries on rather than failing, which is cloud-nuke's call.
	require.NoError(t, err)
	assert.Empty(t, found)

	// And it says so, because a project it could not read is not an empty one.
	assert.Contains(t, logged.String(), "left alone",
		"a refused listing must be reported, or a user reads it as a clean project")
	assert.Contains(t, logged.String(), "gcs-bucket")
}

// TestATypeWithNoCreationTimeIsNotTreatedAsAncient covers the case Pub/Sub
// topics are in: the API reports no creation time at all. Passing the zero
// time would make every such resource look older than any cutoff, so a run
// asking for resources older than an hour would sweep up one made a minute
// ago. A nil time is how the config is told there is nothing to filter on.
func TestATypeWithNoCreationTimeIsNotTreatedAsAncient(t *testing.T) {
	timeless := map[string]any{"name": "demo-timeless", "labels": map[string]any{}}
	api := &bucketAPI{buckets: []map[string]any{timeless}}

	olderThan := time.Now().Add(-time.Hour)
	r := generatedBuckets(t, api, config.ResourceType{
		ExcludeRule: config.FilterRule{TimeAfter: &olderThan},
	})

	found := listThrough(t, r)

	assert.Empty(t, found,
		"a resource with no creation time must not be swept up by an age filter")
}

// stopAfterFirstRead stops the run once one listing has been read in full,
// which is how a partial listing comes about: the locations already read
// answered, and the rest never will.
type stopAfterFirstRead struct {
	base    http.RoundTripper
	stop    context.CancelFunc
	stopped bool
}

func (s *stopAfterFirstRead) RoundTrip(r *http.Request) (*http.Response, error) {
	answer, err := s.base.RoundTrip(r)
	if err != nil || s.stopped {
		return answer, err
	}
	s.stopped = true
	answer.Body = &stopOnClose{ReadCloser: answer.Body, stop: s.stop}
	return answer, nil
}

type stopOnClose struct {
	io.ReadCloser
	stop context.CancelFunc
}

func (s *stopOnClose) Close() error {
	err := s.ReadCloser.Close()
	s.stop()
	return err
}

// TestAStoppedRunDoesNotReportAPartialListing is the difference between a
// location that refused and an operator who stopped the run. The first is a
// gap to warn about and carry on from. The second must not reach a nuke,
// because the listing it would act on is only as long as the run happened to
// get before it was stopped.
func TestAStoppedRunDoesNotReportAPartialListing(t *testing.T) {
	api := &bucketAPI{buckets: []map[string]any{bucket("demo-assets", 48*time.Hour)}}
	server := api.serve(t)

	stopped, stop := context.WithCancel(context.Background())
	client := &http.Client{Transport: &stopAfterFirstRead{base: server.Client().Transport, stop: stop}}

	r := NewGeneratedResource(Generated{
		TypeName:      "gcs-bucket",
		Descriptor:    descriptor(t),
		BaseURL:       server.URL,
		NewHTTPClient: func(context.Context) (*http.Client, error) { return client, nil },
		ConfigGetter:  func(config.Config) config.ResourceType { return config.ResourceType{} },
		// Two projects, so one is read before the run is stopped and the
		// other never is, which is the only way a partial listing exists.
		ScopeOf: func(GcpConfig) generic.Scope {
			return generic.Scope{"project": {"demo", "demo-two"}}
		},
		LocationScope: LocationAppliedByLister,
	})
	r.Init(GcpConfig{ProjectID: "demo"})

	found, err := r.GetAndSetIdentifiers(stopped, config.Config{})
	require.Error(t, err, "a stopped run must not read as a complete listing")
	assert.Empty(t, found, "a stopped run must offer nothing to nuke")
}

// TestATagFromTheGeneratedListingReachesTheFilter pins that the labels the
// descriptor named arrive as tags, which is what a tag-based exclusion runs
// on. Without them every exclusion silently matches nothing.
func TestATagFromTheGeneratedListingReachesTheFilter(t *testing.T) {
	api := &bucketAPI{buckets: []map[string]any{
		bucket("demo-assets", 48*time.Hour),
		bucket("demo-logs", 48*time.Hour),
	}}
	api.buckets[1]["labels"] = map[string]any{"owner": "platform"}

	excluded := config.ResourceType{ExcludeRule: config.FilterRule{
		Tags: map[string]config.Expression{"owner": {RE: *regexp.MustCompile("^team$")}},
	}}

	found := listThrough(t, generatedBuckets(t, api, excluded))
	require.Len(t, found, 1, "the bucket labelled owner=team should have been excluded")
	assert.Contains(t, found[0], "demo-logs")
}
