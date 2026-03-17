package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	// firstSeenLabelKey is the GCP label key used to track when cloud-nuke first discovered a topic.
	// GCP label keys must match [a-z][a-z0-9_-]{0,62}.
	firstSeenLabelKey = "cloud-nuke-first-seen"
)

// NewPubSubTopics creates a new Pub/Sub topic resource using the generic resource pattern.
// Pub/Sub topics are project-scoped; the API does not expose location info on topics,
// so --region/--exclude-region do not filter Pub/Sub results.
func NewPubSubTopics() GcpResource {
	return NewGcpResource(&resource.Resource[*pubsub.PublisherClient]{
		ResourceTypeName: "gcp-pubsub-topic",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*pubsub.PublisherClient], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			r.Scope.Locations = cfg.Locations
			r.Scope.ExcludeLocations = cfg.ExcludeLocations
			client, err := pubsub.NewPublisherClient(context.Background())
			if err != nil {
				panic(fmt.Sprintf("failed to create Pub/Sub publisher client: %v", err))
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpPubSubTopic
		},
		Lister: listPubSubTopics,
		Nuker:  resource.SequentialDeleter(deletePubSubTopic),
	})
}

// listPubSubTopics retrieves all Pub/Sub topics in the project that match the config filters.
//
// Since the Pub/Sub API does not expose a creation timestamp for topics, this function
// uses a first-seen label approach similar to AWS:
//   - On first discovery, a "cloud-nuke-first-seen" label is set on the topic with the
//     current Unix epoch timestamp as the value.
//   - On subsequent runs, the stored timestamp is used for --older-than / --newer-than filtering.
//   - Topics are never deleted on the first run when time-based filters are active.
func listPubSubTopics(ctx context.Context, client *pubsub.PublisherClient, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	excludeFirstSeen, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, fmt.Errorf("unable to read exclude-first-seen-tag from context: %w", err)
	}

	var result []*string

	req := &pubsubpb.ListTopicsRequest{
		Project: fmt.Sprintf("projects/%s", scope.ProjectID),
	}

	it := client.ListTopics(ctx, req)
	for {
		topic, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing Pub/Sub topics: %w", err)
		}

		// topic.Name format: projects/{project}/topics/{topic}
		shortName := topic.Name[strings.LastIndex(topic.Name, "/")+1:]

		labels := topic.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}

		// Get or create the first-seen timestamp via label.
		// Returns nil if --exclude-first-seen-tag is set (time filter is skipped).
		// On failure, log and skip this topic rather than aborting the entire listing —
		// a label update error on one topic should not prevent other topics from being processed.
		firstSeenTime, err := getOrCreatePubSubFirstSeen(ctx, client, topic, labels, excludeFirstSeen)
		if err != nil {
			logging.Errorf("Unable to get or set first-seen label for Pub/Sub topic %s: %v", topic.Name, err)
			continue
		}

		resourceValue := config.ResourceValue{
			Name: &shortName,
			Time: firstSeenTime,
			Tags: labels,
		}

		if cfg.ShouldInclude(resourceValue) {
			name := topic.Name
			result = append(result, &name)
		}
	}

	return result, nil
}

// getOrCreatePubSubFirstSeen checks for the first-seen label on a topic.
// If it exists, it parses and returns the stored timestamp.
// If not, it sets the label with the current Unix epoch and returns the current time.
// GCP label values only support lowercase letters, digits, hyphens, and underscores,
// so Unix epoch seconds (e.g. "1705312200") are used instead of RFC3339.
func getOrCreatePubSubFirstSeen(ctx context.Context, client *pubsub.PublisherClient, topic *pubsubpb.Topic, labels map[string]string, excludeFirstSeen bool) (*time.Time, error) {
	if excludeFirstSeen {
		return nil, nil
	}

	if val, ok := labels[firstSeenLabelKey]; ok {
		epoch, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first-seen label value %q: %w", val, err)
		}
		t := time.Unix(epoch, 0).UTC()
		return &t, nil
	}

	// Label not set — set it now with the current Unix epoch
	now := time.Now().UTC()
	newLabels := make(map[string]string, len(labels)+1)
	for k, v := range labels {
		newLabels[k] = v
	}
	newLabels[firstSeenLabelKey] = strconv.FormatInt(now.Unix(), 10)

	_, err := client.UpdateTopic(ctx, &pubsubpb.UpdateTopicRequest{
		Topic: &pubsubpb.Topic{
			Name:   topic.Name,
			Labels: newLabels,
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"labels"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set first-seen label on topic %s: %w", topic.Name, err)
	}

	return &now, nil
}

// deletePubSubTopic deletes a single Pub/Sub topic.
func deletePubSubTopic(ctx context.Context, client *pubsub.PublisherClient, name *string) error {
	topicName := *name

	req := &pubsubpb.DeleteTopicRequest{
		Topic: topicName,
	}

	if err := client.DeleteTopic(ctx, req); err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Pub/Sub topic %s already deleted, skipping", topicName)
			return nil
		}
		return fmt.Errorf("error deleting Pub/Sub topic %s: %w", topicName, err)
	}

	logging.Debugf("Deleted Pub/Sub topic: %s", topicName)
	return nil
}
