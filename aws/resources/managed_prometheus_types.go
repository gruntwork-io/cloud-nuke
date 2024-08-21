package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ManagedPrometheusAPI interface {
	ListWorkspaces(ctx context.Context, input *amp.ListWorkspacesInput, f ...func(*amp.Options)) (*amp.ListWorkspacesOutput, error)
	DeleteWorkspace(ctx context.Context, params *amp.DeleteWorkspaceInput, optFns ...func(*amp.Options)) (*amp.DeleteWorkspaceOutput, error)
}

type ManagedPrometheus struct {
	BaseAwsResource
	Client     ManagedPrometheusAPI
	Region     string
	WorkSpaces []string
}

func (a *ManagedPrometheus) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ManagedPrometheus
}

func (a *ManagedPrometheus) InitV2(cfg aws.Config) {
	a.Client = amp.NewFromConfig(cfg)
}

func (a *ManagedPrometheus) IsUsingV2() bool { return true }

func (a *ManagedPrometheus) ResourceName() string { return "managed-prometheus" }

func (a *ManagedPrometheus) ResourceIdentifiers() []string { return a.WorkSpaces }

func (a *ManagedPrometheus) MaxBatchSize() int {
	return 100
}

func (a *ManagedPrometheus) Nuke(identifiers []string) error {
	if err := a.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (a *ManagedPrometheus) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := a.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	a.WorkSpaces = aws.ToStringSlice(identifiers)
	return a.WorkSpaces, nil
}
