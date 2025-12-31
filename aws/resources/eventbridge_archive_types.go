package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EventBridgeArchiveAPI interface {
	ListArchives(ctx context.Context, params *eventbridge.ListArchivesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListArchivesOutput, error)
	DeleteArchive(ctx context.Context, params *eventbridge.DeleteArchiveInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteArchiveOutput, error)
}

type EventBridgeArchive struct {
	BaseAwsResource
	Client EventBridgeArchiveAPI
	Region string
	Rules  []string
}

func (eba *EventBridgeArchive) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EventBridgeArchive
}

func (eba *EventBridgeArchive) Init(cfg aws.Config) {
	eba.Client = eventbridge.NewFromConfig(cfg)
}

func (eba *EventBridgeArchive) ResourceName() string { return "event-bridge-archive" }

func (eba *EventBridgeArchive) ResourceIdentifiers() []string { return eba.Rules }

func (eba *EventBridgeArchive) MaxBatchSize() int {
	return 100
}

func (eba *EventBridgeArchive) Nuke(ctx context.Context, identifiers []string) error {
	if err := eba.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (eba *EventBridgeArchive) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := eba.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	eba.Rules = aws.ToStringSlice(identifiers)
	return eba.Rules, nil
}
