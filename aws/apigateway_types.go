package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGateway struct {
	Ids []string
}

func (apigateway ApiGateway) ResourceName() string {
	return "apigateway"
}

func (apigateway ApiGateway) ResourceIdentifiers() []string {
	return apigateway.Ids
}

func (apigateway ApiGateway) MaxBatchSize() int {
	return 10
}

func (apigateway ApiGateway) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAPIGateways(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

type TooManyApiGatewayErr struct{}

func (err TooManyApiGatewayErr) Error() string {
	return "Too many Api Gateways requested at once."
}
