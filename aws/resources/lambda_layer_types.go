package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LambdaLayers struct {
	BaseAwsResource
	Client              lambdaiface.LambdaAPI
	Region              string
	LambdaFunctionNames []string
}

func (lf *LambdaLayers) Init(session *session.Session) {
	lf.Client = lambda.New(session)
}

func (lf *LambdaLayers) ResourceName() string {
	return "lambda_layer"
}

// ResourceIdentifiers - The names of the lambda functions
func (lf *LambdaLayers) ResourceIdentifiers() []string {
	return lf.LambdaFunctionNames
}

func (lf *LambdaLayers) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (lf *LambdaLayers) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := lf.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	lf.LambdaFunctionNames = awsgo.StringValueSlice(identifiers)
	return lf.LambdaFunctionNames, nil
}

// Nuke - nuke 'em all!!!
func (lf *LambdaLayers) Nuke(identifiers []string) error {
	if err := lf.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type LambdaVersionDeleteError struct {
	name string
}

func (e LambdaVersionDeleteError) Error() string {
	return "Lambda Function:" + e.name + "was not deleted"
}
