package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// scope - represents all scopes
type EC2IpamScopes struct {
	BaseAwsResource
	Client    ec2iface.EC2API
	Region    string
	ScopreIDs []string
}

func (scope *EC2IpamScopes) Init(session *session.Session) {
	scope.Client = ec2.New(session)
}

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

	scope.ScopreIDs = awsgo.StringValueSlice(identifiers)
	return scope.ScopreIDs, nil
}

// Nuke - nuke 'em all!!!
func (scope *EC2IpamScopes) Nuke(identifiers []string) error {
	if err := scope.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
