package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGateway struct {
	Client apigatewayiface.APIGatewayAPI
	Region string
	Ids    []string
}

func (gateway *ApiGateway) Init(session *session.Session) {
	gateway.Client = apigateway.New(session)
}

func (gateway *ApiGateway) ResourceName() string {
	return "apigateway"
}

func (gateway *ApiGateway) ResourceIdentifiers() []string {
	return gateway.Ids
}

func (gateway *ApiGateway) MaxBatchSize() int {
	return 10
}

func (gateway *ApiGateway) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := gateway.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	gateway.Ids = awsgo.StringValueSlice(identifiers)
	return gateway.Ids, nil
}

func (gateway *ApiGateway) Nuke(identifiers []string) error {
	if err := gateway.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type TooManyApiGatewayErr struct{}

func (err TooManyApiGatewayErr) Error() string {
	return "Too many Api Gateways requested at once."
}
