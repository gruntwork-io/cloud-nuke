package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type InternetGateway struct {
	BaseAwsResource
	Client     ec2iface.EC2API
	Region     string
	GatewayIds []string
}

func (igw *InternetGateway) Init(session *session.Session) {
	igw.BaseAwsResource.Init(session)
	igw.Client = ec2.New(session)
}

func (igw *InternetGateway) ResourceName() string {
	return "internet-gateway"
}

func (igw *InternetGateway) ResourceIdentifiers() []string {
	return igw.GatewayIds
}

func (igw *InternetGateway) MaxBatchSize() int {
	return 50
}

func (igw *InternetGateway) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := igw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	igw.GatewayIds = aws.StringValueSlice(identifiers)
	return igw.GatewayIds, nil
}

func (igw *InternetGateway) Nuke(identifiers []string) error {
	if err := igw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
