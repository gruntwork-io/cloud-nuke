package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/apigatewayv2/apigatewayv2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGatewayV2 struct {
	Client apigatewayv2iface.ApiGatewayV2API
	Region string
	Ids    []string
}

func (gw *ApiGatewayV2) Init(session *session.Session) {
	gw.Client = apigatewayv2.New(session)
}

func (gw *ApiGatewayV2) ResourceName() string {
	return "apigatewayv2"
}

func (gw *ApiGatewayV2) ResourceIdentifiers() []string {
	return gw.Ids
}

func (gw *ApiGatewayV2) MaxBatchSize() int {
	return 10
}

func (gw *ApiGatewayV2) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := gw.getAll(configObj)
	if err != nil {
		return nil, err
	}

	gw.Ids = awsgo.StringValueSlice(identifiers)
	return gw.Ids, nil
}

func (gw *ApiGatewayV2) Nuke(identifiers []string) error {
	if err := gw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type TooManyApiGatewayV2Err struct{}

func (err TooManyApiGatewayV2Err) Error() string {
	return "Too many Api Gateways requested at once."
}
