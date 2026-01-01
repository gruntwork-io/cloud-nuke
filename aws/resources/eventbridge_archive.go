package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EventBridgeArchiveAPI defines the interface for EventBridge Archive operations.
type EventBridgeArchiveAPI interface {
	ListArchives(ctx context.Context, params *eventbridge.ListArchivesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListArchivesOutput, error)
	DeleteArchive(ctx context.Context, params *eventbridge.DeleteArchiveInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteArchiveOutput, error)
}

// NewEventBridgeArchive creates a new EventBridgeArchive resource using the generic resource pattern.
func NewEventBridgeArchive() AwsResource {
	return NewAwsResource(&resource.Resource[EventBridgeArchiveAPI]{
		ResourceTypeName: "event-bridge-archive",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EventBridgeArchiveAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = eventbridge.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EventBridgeArchive
		},
		Lister: listEventBridgeArchives,
		Nuker:  resource.SimpleBatchDeleter(deleteEventBridgeArchive),
	})
}

// listEventBridgeArchives retrieves all EventBridge Archives that match the config filters.
func listEventBridgeArchives(ctx context.Context, client EventBridgeArchiveAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	params := eventbridge.ListArchivesInput{}
	hasMorePages := true

	for hasMorePages {
		archives, err := client.ListArchives(ctx, &params)
		if err != nil {
			logging.Debugf("[Event Bridge Archives] Failed to list archives: %s", err)
			return nil, err
		}

		for _, archive := range archives.Archives {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: archive.ArchiveName,
				Time: archive.CreationTime,
			}) {
				identifiers = append(identifiers, archive.ArchiveName)
			}
		}

		params.NextToken = archives.NextToken
		hasMorePages = params.NextToken != nil
	}

	return identifiers, nil
}

// deleteEventBridgeArchive deletes a single EventBridge Archive.
func deleteEventBridgeArchive(ctx context.Context, client EventBridgeArchiveAPI, archiveName *string) error {
	_, err := client.DeleteArchive(ctx, &eventbridge.DeleteArchiveInput{
		ArchiveName: archiveName,
	})
	return err
}
