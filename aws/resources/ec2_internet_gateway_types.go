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
	Client        ec2iface.EC2API
	Region        string
	GatewayIds    []string
	GatewayVPCMap map[string]string
}

func (igw *InternetGateway) Init(session *session.Session) {
	igw.BaseAwsResource.Init(session)
	igw.Client = ec2.New(session)

	// Since the nuking of the internet gateway requires the VPC ID, and to avoid redundant API calls for this information within the nuke method,
	// we utilize the getAll method to retrieve it.
	// This map is used to store the information and access the value within the nuke method.
	igw.GatewayVPCMap = make(map[string]string)
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

func (igw *InternetGateway) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.InternetGateway
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
