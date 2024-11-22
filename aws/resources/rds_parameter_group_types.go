package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RdsParameterGroupAPI interface {
	DeleteDBParameterGroup(ctx context.Context, params *rds.DeleteDBParameterGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBParameterGroupOutput, error)
	DescribeDBParameterGroups(ctx context.Context, params *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error)
}
type RdsParameterGroup struct {
	BaseAwsResource
	Client     RdsParameterGroupAPI
	Region     string
	GroupNames []string
}

func (pg *RdsParameterGroup) InitV2(cfg aws.Config) {
	pg.Client = rds.NewFromConfig(cfg)
}

func (pg *RdsParameterGroup) IsUsingV2() bool { return true }

func (pg *RdsParameterGroup) ResourceName() string {
	return "rds-parameter-group"
}

// ResourceIdentifiers - The names of the rds parameter group
func (pg *RdsParameterGroup) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg *RdsParameterGroup) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (pg *RdsParameterGroup) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.RdsParameterGroup
}

func (pg *RdsParameterGroup) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pg.GroupNames = aws.ToStringSlice(identifiers)
	return pg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (pg *RdsParameterGroup) Nuke(identifiers []string) error {
	if err := pg.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
