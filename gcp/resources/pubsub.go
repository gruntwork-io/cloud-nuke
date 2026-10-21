package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pubsub "cloud.google.com/go/pubsub/v2/apiv1"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
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
	// firstSeenLabelKey records when cloud-nuke first saw a topic, because the
	// Pub/Sub API reports no creation time for one. GCP label keys must match
	// [a-z][a-z0-9_-]{0,62}.
	firstSeenLabelKey = "cloud-nuke-first-seen"
)

// PubSubAPI is the part of the Pub/Sub API this resource uses. The SDK answers
// a list with an iterator a test cannot build, so the iteration lives in the
// adapter below and the first-seen logic stays testable.
type PubSubAPI interface {
	// ListTopics returns every topic in the project.
	ListTopics(ctx context.Context, project string) ([]PubSubTopic, error)

	// SetTopicLabels replaces the labels on one topic, which is how the
	// first-seen timestamp is recorded.
	SetTopicLabels(ctx context.Context, name string, labels map[string]string) error

	// DeleteTopic deletes one topic. Deleting one that is already gone is not
	// an error.
	DeleteTopic(ctx context.Context, name string) error
}

// PubSubTopic is what this resource needs about one topic.
type PubSubTopic struct {
	// Name is projects/{project}/topics/{topic}.
	Name   string
	Labels map[string]string
}

// NewPubSubTopics creates a new Pub/Sub topic resource using the generic resource pattern.
func NewPubSubTopics() GcpResource {
	// A Pub/Sub topic is project-global: the API reports no location for it.
	return NewGcpResourceWithLocationScope(&resource.Resource[PubSubAPI]{
		ResourceTypeName: "gcp-pubsub-topic",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[PubSubAPI], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := pubsub.NewTopicAdminClient(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Pub/Sub topic admin client: %w", err)
				return
			}
			r.Client = pubSubClient{client: client}
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpPubSubTopic
		},
		Lister: listPubSubTopics,
		Nuker:  resource.SequentialDeleter(deletePubSubTopic),
	}, LocationNone)
}

// listPubSubTopics retrieves all Pub/Sub topics in the project that match the
// config filters.
//
// The Pub/Sub API reports no creation time for a topic, so cloud-nuke records
// when it first saw one in a label, the way it does with a tag on AWS. On the
// first run the label is written and the topic's age is the moment it was seen,
// so a topic is never deleted by --older-than on the run that discovered it.
func listPubSubTopics(ctx context.Context, client PubSubAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	excludeFirstSeen, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, fmt.Errorf("unable to read exclude-first-seen-tag from context: %w", err)
	}

	topics, err := client.ListTopics(ctx, fmt.Sprintf("projects/%s", scope.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("error listing Pub/Sub topics: %w", err)
	}

	var result []*string
	for _, topic := range topics {
		shortName := topic.Name[strings.LastIndex(topic.Name, "/")+1:]

		labels := topic.Labels
		if labels == nil {
			labels = map[string]string{}
		}

		// A label write failing on one topic costs that topic rather than the
		// whole listing, since the others can still be filtered and deleted.
		firstSeen, err := getOrCreatePubSubFirstSeen(ctx, client, topic.Name, labels, excludeFirstSeen)
		if err != nil {
			logging.Errorf("Unable to get or set first-seen label for Pub/Sub topic %s: %v", topic.Name, err)
			continue
		}

		if cfg.ShouldInclude(config.ResourceValue{Name: &shortName, Time: firstSeen, Tags: labels}) {
			name := topic.Name
			result = append(result, &name)
		}
	}

	return result, nil
}

// getOrCreatePubSubFirstSeen reads the first-seen label, or writes it with the
// current time when the topic does not carry one yet. It returns nil when the
// caller asked for the label to be ignored, which switches the time filters off
// for this resource type. The value is Unix epoch seconds because a GCP label
// value accepts only lowercase letters, digits, hyphens and underscores.
func getOrCreatePubSubFirstSeen(ctx context.Context, client PubSubAPI, name string, labels map[string]string, excludeFirstSeen bool) (*time.Time, error) {
	if excludeFirstSeen {
		return nil, nil
	}

	if value, ok := labels[firstSeenLabelKey]; ok {
		epoch, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first-seen label value %q: %w", value, err)
		}
		seen := time.Unix(epoch, 0).UTC()
		return &seen, nil
	}

	now := time.Now().UTC()
	withFirstSeen := make(map[string]string, len(labels)+1)
	for key, value := range labels {
		withFirstSeen[key] = value
	}
	withFirstSeen[firstSeenLabelKey] = strconv.FormatInt(now.Unix(), 10)

	if err := client.SetTopicLabels(ctx, name, withFirstSeen); err != nil {
		return nil, fmt.Errorf("failed to set first-seen label on topic %s: %w", name, err)
	}

	return &now, nil
}

// deletePubSubTopic deletes a single Pub/Sub topic.
func deletePubSubTopic(ctx context.Context, client PubSubAPI, name *string) error {
	if err := client.DeleteTopic(ctx, *name); err != nil {
		return fmt.Errorf("error deleting Pub/Sub topic %s: %w", *name, err)
	}

	logging.Debugf("Deleted Pub/Sub topic: %s", *name)
	return nil
}

// pubSubClient drives the SDK: the iterator and the field mask.
type pubSubClient struct {
	client *pubsub.TopicAdminClient
}

func (c pubSubClient) ListTopics(ctx context.Context, project string) ([]PubSubTopic, error) {
	var found []PubSubTopic

	it := c.client.ListTopics(ctx, &pubsubpb.ListTopicsRequest{Project: project})
	for {
		topic, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return found, nil
		}
		if err != nil {
			return nil, err
		}
		found = append(found, PubSubTopic{Name: topic.Name, Labels: topic.GetLabels()})
	}
}

func (c pubSubClient) SetTopicLabels(ctx context.Context, name string, labels map[string]string) error {
	_, err := c.client.UpdateTopic(ctx, &pubsubpb.UpdateTopicRequest{
		Topic:      &pubsubpb.Topic{Name: name, Labels: labels},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
	})
	return err
}

func (c pubSubClient) DeleteTopic(ctx context.Context, name string) error {
	if err := c.client.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{Topic: name}); err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Pub/Sub topic %s already deleted, skipping", name)
			return nil
		}
		return err
	}
	return nil
}

var _ PubSubAPI = pubSubClient{}
