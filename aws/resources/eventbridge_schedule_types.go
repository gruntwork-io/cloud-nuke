package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EventBridgeScheduleAPI interface {
	DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
	ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error)
}

type EventBridgeSchedule struct {
	BaseAwsResource
	Client    EventBridgeScheduleAPI
	Region    string
	Schedules []string
}

func (sch *EventBridgeSchedule) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EventBridgeSchedule
}

func (sch *EventBridgeSchedule) InitV2(cfg aws.Config) {
	sch.Client = scheduler.NewFromConfig(cfg)
}

func (sch *EventBridgeSchedule) IsUsingV2() bool { return true }

func (sch *EventBridgeSchedule) ResourceName() string { return "event-bridge-schedule" }

func (sch *EventBridgeSchedule) ResourceIdentifiers() []string { return sch.Schedules }

func (sch *EventBridgeSchedule) MaxBatchSize() int {
	return 100
}

func (sch *EventBridgeSchedule) Nuke(identifiers []string) error {
	if err := sch.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (sch *EventBridgeSchedule) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := sch.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	sch.Schedules = aws.ToStringSlice(identifiers)
	return sch.Schedules, nil
}
