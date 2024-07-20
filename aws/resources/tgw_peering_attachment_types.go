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
type TransitGatewayPeeringAttachment struct {
	BaseAwsResource
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

func (tgpa *TransitGatewayPeeringAttachment) Init(session *session.Session) {
	tgpa.Client = ec2.New(session)
}

func (tgpa *TransitGatewayPeeringAttachment) ResourceName() string {
	return "transit-gateway-peering-attachment"
}

func (tgpa *TransitGatewayPeeringAttachment) MaxBatchSize() int {
	return maxBatchSize
}

func (tgpa *TransitGatewayPeeringAttachment) ResourceIdentifiers() []string {
	return tgpa.Ids
}

func (tgpa *TransitGatewayPeeringAttachment) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.TransitGateway
}

func (tgpa *TransitGatewayPeeringAttachment) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgpa.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgpa.Ids = awsgo.StringValueSlice(identifiers)
	return tgpa.Ids, nil
}

func (tgpa *TransitGatewayPeeringAttachment) Nuke(identifiers []string) error {
	if err := tgpa.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
