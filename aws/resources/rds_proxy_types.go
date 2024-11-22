package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RdsProxyAPI interface {
	DescribeDBProxies(ctx context.Context, params *rds.DescribeDBProxiesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBProxiesOutput, error)
	DeleteDBProxy(ctx context.Context, params *rds.DeleteDBProxyInput, optFns ...func(*rds.Options)) (*rds.DeleteDBProxyOutput, error)
}
type RdsProxy struct {
	BaseAwsResource
	Client     RdsProxyAPI
	Region     string
	GroupNames []string
}

func (pg *RdsProxy) InitV2(cfg aws.Config) {
	pg.Client = rds.NewFromConfig(cfg)
}

func (pg *RdsProxy) IsUsingV2() bool { return true }

func (pg *RdsProxy) ResourceName() string {
	return "rds-proxy"
}

// ResourceIdentifiers - The names of the rds parameter group
func (pg *RdsProxy) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg *RdsProxy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (pg *RdsProxy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.RdsProxy
}

func (pg *RdsProxy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pg.GroupNames = awsgo.ToStringSlice(identifiers)
	return pg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (pg *RdsProxy) Nuke(identifiers []string) error {
	if err := pg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
