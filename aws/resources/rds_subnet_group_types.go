package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBSubnetGroupsAPI interface {
	DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error)
	DeleteDBSubnetGroup(ctx context.Context, params *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error)
}

type DBSubnetGroups struct {
	BaseAwsResource
	Client        DBSubnetGroupsAPI
	Region        string
	InstanceNames []string
}

func (dsg *DBSubnetGroups) InitV2(cfg aws.Config) {
	dsg.Client = rds.NewFromConfig(cfg)
}

func (dsg *DBSubnetGroups) ResourceName() string {
	return "rds-subnet-group"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (dsg *DBSubnetGroups) ResourceIdentifiers() []string {
	return dsg.InstanceNames
}

func (dsg *DBSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (dsg *DBSubnetGroups) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DBSubnetGroups
}

func (dsg *DBSubnetGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := dsg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	dsg.InstanceNames = aws.ToStringSlice(identifiers)
	return dsg.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (dsg *DBSubnetGroups) Nuke(identifiers []string) error {
	if err := dsg.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
