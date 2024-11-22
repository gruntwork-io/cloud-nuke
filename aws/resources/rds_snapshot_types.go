package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RdsSnapshotAPI interface {
	DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	DeleteDBSnapshot(ctx context.Context, params *rds.DeleteDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error)
}

type RdsSnapshot struct {
	BaseAwsResource
	Client      RdsSnapshotAPI
	Region      string
	Identifiers []string
}

func (snapshot *RdsSnapshot) InitV2(cfg aws.Config) {
	snapshot.Client = rds.NewFromConfig(cfg)
}

func (snapshot *RdsSnapshot) IsUsingV2() bool { return true }

func (snapshot *RdsSnapshot) ResourceName() string {
	return "rds-snapshot"
}

func (snapshot *RdsSnapshot) ResourceIdentifiers() []string {
	return snapshot.Identifiers
}

func (snapshot *RdsSnapshot) MaxBatchSize() int {
	return 49
}

func (snapshot *RdsSnapshot) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.RdsSnapshot
}

func (snapshot *RdsSnapshot) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := snapshot.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	snapshot.Identifiers = aws.ToStringSlice(identifiers)
	return snapshot.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (snapshot *RdsSnapshot) Nuke(identifiers []string) error {
	if err := snapshot.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
