package resources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// fakeStorageAPI answers the one call listGCSBuckets makes, so the filter can
// be tested without credentials or a real bucket. A GCS bucket's name carries
// no location, which is why this lister filters itself instead of letting the
// adapter do it, and why it needs its own test.
func fakeStorageAPI(t *testing.T, buckets string) *storage.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, buckets)
	}))
	t.Cleanup(server.Close)

	client, err := storage.NewClient(context.Background(),
		option.WithEndpoint(server.URL),
		option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func TestListGCSBucketsAppliesTheLocationFilter(t *testing.T) {
	t.Parallel()

	// GCS reports a location in upper case, and a multi-region bucket reports
	// the multi-region rather than any region inside it.
	answer := `{"kind":"storage#buckets","items":[
		{"name":"in-central","location":"US-CENTRAL1","timeCreated":"2026-01-01T00:00:00Z"},
		{"name":"in-west4","location":"EUROPE-WEST4","timeCreated":"2026-01-01T00:00:00Z"},
		{"name":"multi-region","location":"US","timeCreated":"2026-01-01T00:00:00Z"}
	]}`

	for _, tc := range []struct {
		name     string
		cfg      GcpConfig
		expected []string
	}{
		{
			name:     "no filter keeps every bucket",
			cfg:      GcpConfig{ProjectID: "p"},
			expected: []string{"in-central", "in-west4", "multi-region"},
		},
		{
			name:     "an included location keeps only the buckets in it",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}},
			expected: []string{"in-central"},
		},
		{
			name:     "the filter ignores case, which GCS reports in upper",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"US-CENTRAL1"}},
			expected: []string{"in-central"},
		},
		{
			name:     "an excluded location drops its buckets and keeps the rest",
			cfg:      GcpConfig{ProjectID: "p", ExcludeLocations: []string{"us-central1"}},
			expected: []string{"in-west4", "multi-region"},
		},
		{
			name: "a multi-region bucket answers to its multi-region and not to a region inside it",
			cfg:  GcpConfig{ProjectID: "p", Locations: []string{"us"}},
			// US holds us-central1, but GCS reports the bucket's location as US,
			// so asking for us-central1 does not reach it and asking for US does.
			expected: []string{"multi-region"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakeStorageAPI(t, answer)
			identifiers, err := listGCSBuckets(context.Background(), client,
				resource.Scope{ProjectID: "p"}, config.ResourceType{}, tc.cfg)
			require.NoError(t, err)

			var names []string
			for _, identifier := range identifiers {
				names = append(names, *identifier)
			}
			assert.Equal(t, tc.expected, names)
		})
	}
}
