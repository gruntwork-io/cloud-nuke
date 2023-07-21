package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewayv2/apigatewayv2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGatewayV2 struct {
	Client apigatewayv2iface.ApiGatewayV2API
	Region string
	Ids    []string
}

func (apigateway ApiGatewayV2) ResourceName() string {
	return "apigatewayv2"
}

func (apigateway ApiGatewayV2) ResourceIdentifiers() []string {
	return apigateway.Ids
}

func (apigateway ApiGatewayV2) MaxBatchSize() int {
	return 10
}

func (apigateway ApiGatewayV2) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAPIGatewaysV2(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type TooManyApiGatewayV2Err struct{}

func (err TooManyApiGatewayV2Err) Error() string {
	return "Too many Api Gateways requested at once."
}
