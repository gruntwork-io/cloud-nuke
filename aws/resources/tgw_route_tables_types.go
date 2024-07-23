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

// TransitGatewaysRouteTables - represents all transit gateways route tables
type TransitGatewaysRouteTables struct {
	BaseAwsResource
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

func (tgw *TransitGatewaysRouteTables) Init(session *session.Session) {
	tgw.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (tgw *TransitGatewaysRouteTables) ResourceName() string {
	return "transit-gateway-route-table"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw *TransitGatewaysRouteTables) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The arns of the transit gateways route tables
func (tgw *TransitGatewaysRouteTables) ResourceIdentifiers() []string {
	return tgw.Ids
}

func (tgw *TransitGatewaysRouteTables) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgw.Ids = awsgo.StringValueSlice(identifiers)
	return tgw.Ids, nil
}

// Nuke - nuke 'em all!!!
func (tgw *TransitGatewaysRouteTables) Nuke(identifiers []string) error {
	if err := tgw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
