package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2IpamScopesAPI interface {
	DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error)
	DeleteIpamScope(ctx context.Context, params *ec2.DeleteIpamScopeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamScopeOutput, error)
}

// EC2IpamScopes scope - represents all scopes
type EC2IpamScopes struct {
	BaseAwsResource
	Client    EC2IpamScopesAPI
	Region    string
	ScopreIDs []string
}

func (scope *EC2IpamScopes) InitV2(cfg aws.Config) {
	scope.Client = ec2.NewFromConfig(cfg)
}

func (scope *EC2IpamScopes) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (scope *EC2IpamScopes) ResourceName() string {
	return "ipam-scope"
}

func (scope *EC2IpamScopes) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the scopes
func (scope *EC2IpamScopes) ResourceIdentifiers() []string {
	return scope.ScopreIDs
}

func (scope *EC2IpamScopes) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2IPAMScope
}

func (scope *EC2IpamScopes) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := scope.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	scope.ScopreIDs = aws.ToStringSlice(identifiers)
	return scope.ScopreIDs, nil
}

// Nuke - nuke 'em all!!!
func (scope *EC2IpamScopes) Nuke(identifiers []string) error {
	if err := scope.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
