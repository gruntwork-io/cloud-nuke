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

// NatGateways - represents all AWS secrets manager secrets that should be deleted.
type NatGateways struct {
	BaseAwsResource
	Client        ec2iface.EC2API
	Region        string
	NatGatewayIDs []string
}

func (ngw *NatGateways) Init(session *session.Session) {
	ngw.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (ngw *NatGateways) ResourceName() string {
	return "nat-gateway"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ngw *NatGateways) ResourceIdentifiers() []string {
	return ngw.NatGatewayIDs
}

func (ngw *NatGateways) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (ngw *NatGateways) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NatGateway
}

func (ngw *NatGateways) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ngw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ngw.NatGatewayIDs = awsgo.StringValueSlice(identifiers)
	return ngw.NatGatewayIDs, nil
}

// Nuke - nuke 'em all!!!
func (ngw *NatGateways) Nuke(identifiers []string) error {
	if err := ngw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
