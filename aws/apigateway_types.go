package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGateway struct {
	Client apigatewayiface.APIGatewayAPI
	Region string
	Ids    []string
}

func (gateway ApiGateway) ResourceName() string {
	return "apigateway"
}

func (gateway ApiGateway) ResourceIdentifiers() []string {
	return gateway.Ids
}

func (gateway ApiGateway) MaxBatchSize() int {
	return 10
}

func (gateway ApiGateway) Nuke(session *session.Session, identifiers []string) error {
	// TODO(james): stop passing in session argument as it is included as part of the gateway struct.
	if err := gateway.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type TooManyApiGatewayErr struct{}

func (err TooManyApiGatewayErr) Error() string {
	return "Too many Api Gateways requested at once."
}
