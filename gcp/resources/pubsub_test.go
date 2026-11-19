package resources

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePubSub answers with the topics a test names and records the labels it
// was asked to write.
type fakePubSub struct {
	topics       []PubSubTopic
	labelErr     error
	labelsSet    map[string]map[string]string
	deleted      []string
	projectAsked string
}

func (f *fakePubSub) ListTopics(ctx context.Context, project string) ([]PubSubTopic, error) {
	f.projectAsked = project
	return f.topics, nil
}

func (f *fakePubSub) SetTopicLabels(ctx context.Context, name string, labels map[string]string) error {
	if f.labelErr != nil {
		return f.labelErr
	}
	if f.labelsSet == nil {
		f.labelsSet = map[string]map[string]string{}
	}
	f.labelsSet[name] = labels
	return nil
}

func (f *fakePubSub) DeleteTopic(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

// withFirstSeenEnabled is the context the engine builds when the user did not
// pass --exclude-first-seen.
func withFirstSeenEnabled() context.Context {
	return context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
}

func TestPubSubTopics_ResourceName(t *testing.T) {
	assert.Equal(t, "gcp-pubsub-topic", NewPubSubTopics().ResourceName())
}

func TestPubSubTopics_MaxBatchSize(t *testing.T) {
	assert.Equal(t, 50, NewPubSubTopics().MaxBatchSize())
}

func TestListPubSubTopicsWritesFirstSeenOnTheRunThatDiscoversATopic(t *testing.T) {
	t.Parallel()

	client := &fakePubSub{topics: []PubSubTopic{{Name: "projects/p/topics/new"}}}

	before := time.Now().UTC().Add(-time.Second)
	identifiers, err := listPubSubTopics(withFirstSeenEnabled(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})
	require.NoError(t, err)

	// The Pub/Sub API reports no creation time, so the first run records one.
	require.Len(t, identifiers, 1)
	written := client.labelsSet["projects/p/topics/new"]
	require.Contains(t, written, firstSeenLabelKey)

	epoch, err := strconv.ParseInt(written[firstSeenLabelKey], 10, 64)
	require.NoError(t, err)
	assert.WithinDuration(t, before, time.Unix(epoch, 0).UTC(), 10*time.Second)
	assert.Equal(t, "projects/p", client.projectAsked)
}

func TestListPubSubTopicsReadsAFirstSeenLabelItAlreadyWrote(t *testing.T) {
	t.Parallel()

	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour)
	client := &fakePubSub{topics: []PubSubTopic{{
		Name:   "projects/p/topics/old",
		Labels: map[string]string{firstSeenLabelKey: strconv.FormatInt(threeDaysAgo.Unix(), 10)},
	}}}

	// Only topics first seen more than a day ago are included, which the
	// stored label is what answers.
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	identifiers, err := listPubSubTopics(withFirstSeenEnabled(), client,
		resource.Scope{ProjectID: "p"},
		config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: &yesterday}})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/topics/old", *identifiers[0])
	assert.Empty(t, client.labelsSet, "a topic that already carries the label is not written to again")
}

func TestListPubSubTopicsWritesNoLabelWhenTheUserExcludedFirstSeen(t *testing.T) {
	t.Parallel()

	client := &fakePubSub{topics: []PubSubTopic{{Name: "projects/p/topics/new"}}}
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true)

	identifiers, err := listPubSubTopics(ctx, client, resource.Scope{ProjectID: "p"}, config.ResourceType{})
	require.NoError(t, err)

	// --exclude-first-seen means cloud-nuke must not write to the resource.
	require.Len(t, identifiers, 1)
	assert.Empty(t, client.labelsSet)
}

func TestListPubSubTopicsSkipsOneTopicWhenItsLabelCannotBeWritten(t *testing.T) {
	t.Parallel()

	client := &fakePubSub{
		topics: []PubSubTopic{
			{Name: "projects/p/topics/unwritable"},
			{Name: "projects/p/topics/fine", Labels: map[string]string{
				firstSeenLabelKey: strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10),
			}},
		},
		labelErr: errors.New("permission denied"),
	}

	identifiers, err := listPubSubTopics(withFirstSeenEnabled(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	// One topic refusing a label write costs that topic, not the listing.
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/topics/fine", *identifiers[0])
}

func TestListPubSubTopicsRefusesAnUnparseableFirstSeenLabel(t *testing.T) {
	t.Parallel()

	client := &fakePubSub{topics: []PubSubTopic{{
		Name:   "projects/p/topics/corrupt",
		Labels: map[string]string{firstSeenLabelKey: "not-a-number"},
	}}}

	identifiers, err := listPubSubTopics(withFirstSeenEnabled(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	// A label nobody can read is not evidence of age, so the topic is skipped
	// rather than treated as new and deleted.
	require.NoError(t, err)
	assert.Empty(t, identifiers)
}

func TestNukePubSubTopicDeletesByFullName(t *testing.T) {
	t.Parallel()

	client := &fakePubSub{}
	name := "projects/p/topics/one"
	require.NoError(t, deletePubSubTopic(context.Background(), client, &name))
	assert.Equal(t, []string{name}, client.deleted)
}
