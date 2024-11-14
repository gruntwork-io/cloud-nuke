package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/grafana"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type GrafanaAPI interface {
	DeleteWorkspace(ctx context.Context, params *grafana.DeleteWorkspaceInput, optFns ...func(*grafana.Options)) (*grafana.DeleteWorkspaceOutput, error)
	ListWorkspaces(ctx context.Context, params *grafana.ListWorkspacesInput, optFns ...func(*grafana.Options)) (*grafana.ListWorkspacesOutput, error)
}

type Grafana struct {
	BaseAwsResource
	Client     GrafanaAPI
	Region     string
	WorkSpaces []string
}

func (g *Grafana) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ManagedPrometheus
}

func (g *Grafana) InitV2(cfg aws.Config) {
	g.Client = grafana.NewFromConfig(cfg)
}

func (g *Grafana) IsUsingV2() bool { return true }

func (g *Grafana) ResourceName() string { return "grafana" }

func (g *Grafana) ResourceIdentifiers() []string { return g.WorkSpaces }

func (g *Grafana) MaxBatchSize() int {
	return 100
}

func (g *Grafana) Nuke(identifiers []string) error {
	if err := g.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (g *Grafana) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := g.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	g.WorkSpaces = aws.ToStringSlice(identifiers)
	return g.WorkSpaces, nil
}
