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

// TransitGateways - represents all transit gateways
type TransitGateways struct {
	BaseAwsResource
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

func (tgw *TransitGateways) Init(session *session.Session) {
	// to initialize base resource
	// NOTE : This is madatory to initialize the nukables map
	tgw.BaseAwsResource.Init(session)
	tgw.Client = ec2.New(session)

}

// ResourceName - the simple name of the aws resource
func (tgw *TransitGateways) ResourceName() string {
	return "transit-gateway"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw *TransitGateways) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw *TransitGateways) ResourceIdentifiers() []string {
	return tgw.Ids
}

func (tgw *TransitGateways) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgw.Ids = awsgo.StringValueSlice(identifiers)
	return tgw.Ids, nil
}

// Nuke - nuke 'em all!!!
func (tgw *TransitGateways) Nuke(identifiers []string) error {
	if err := tgw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
