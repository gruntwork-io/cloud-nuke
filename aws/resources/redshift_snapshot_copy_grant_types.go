package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RedshiftSnapshotCopyGrantsAPI interface {
	DescribeSnapshotCopyGrants(ctx context.Context, params *redshift.DescribeSnapshotCopyGrantsInput, optFns ...func(*redshift.Options)) (*redshift.DescribeSnapshotCopyGrantsOutput, error)
	DeleteSnapshotCopyGrant(ctx context.Context, params *redshift.DeleteSnapshotCopyGrantInput, optFns ...func(*redshift.Options)) (*redshift.DeleteSnapshotCopyGrantOutput, error)
}

type RedshiftSnapshotCopyGrants struct {
	BaseAwsResource
	Client     RedshiftSnapshotCopyGrantsAPI
	Region     string
	GrantNames []string
}

func (g *RedshiftSnapshotCopyGrants) Init(cfg aws.Config) {
	g.Client = redshift.NewFromConfig(cfg)
}

func (g *RedshiftSnapshotCopyGrants) ResourceName() string {
	return "redshift-snapshot-copy-grant"
}

func (g *RedshiftSnapshotCopyGrants) ResourceIdentifiers() []string {
	return g.GrantNames
}

func (g *RedshiftSnapshotCopyGrants) MaxBatchSize() int {
	return 49
}

func (g *RedshiftSnapshotCopyGrants) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.RedshiftSnapshotCopyGrant
}

func (g *RedshiftSnapshotCopyGrants) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := g.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	g.GrantNames = aws.ToStringSlice(identifiers)
	return g.GrantNames, nil
}

func (g *RedshiftSnapshotCopyGrants) Nuke(ctx context.Context, identifiers []string) error {
	if err := g.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
