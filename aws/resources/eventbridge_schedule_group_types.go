package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EventBridgeScheduleGroupAPI interface {
	DeleteScheduleGroup(ctx context.Context, params *scheduler.DeleteScheduleGroupInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleGroupOutput, error)
	ListScheduleGroups(ctx context.Context, params *scheduler.ListScheduleGroupsInput, optFns ...func(*scheduler.Options)) (*scheduler.ListScheduleGroupsOutput, error)
}

type EventBridgeScheduleGroup struct {
	BaseAwsResource
	Client EventBridgeScheduleGroupAPI
	Region string
	Groups []string
}

func (sch *EventBridgeScheduleGroup) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EventBridgeScheduleGroup
}

func (sch *EventBridgeScheduleGroup) InitV2(cfg aws.Config) {
	sch.Client = scheduler.NewFromConfig(cfg)
}

func (sch *EventBridgeScheduleGroup) IsUsingV2() bool { return true }

func (sch *EventBridgeScheduleGroup) ResourceName() string { return "event-bridge-schedule-group" }

func (sch *EventBridgeScheduleGroup) ResourceIdentifiers() []string { return sch.Groups }

func (sch *EventBridgeScheduleGroup) MaxBatchSize() int {
	return 100
}

func (sch *EventBridgeScheduleGroup) Nuke(identifiers []string) error {
	if err := sch.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (sch *EventBridgeScheduleGroup) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := sch.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	sch.Groups = aws.ToStringSlice(identifiers)
	return sch.Groups, nil
}
