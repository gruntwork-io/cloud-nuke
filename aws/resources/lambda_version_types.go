package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
)

type LambdaFunctionVersions struct {
	Client              lambdaiface.LambdaAPI
	Region              string
	LambdaFunctionNames []string
}

func (lf *LambdaFunctionVersions) Init(session *session.Session) {
	lf.Client = lambda.New(session)
}

func (lf *LambdaFunctionVersions) ResourceName() string {
	return "lambda"
}

// ResourceIdentifiers - The names of the lambda functions
func (lf *LambdaFunctionVersions) ResourceIdentifiers() []string {
	return lf.LambdaFunctionNames
}

func (lf *LambdaFunctionVersions) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (lf *LambdaFunctionVersions) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := lf.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	lf.LambdaFunctionNames = awsgo.StringValueSlice(identifiers)
	return lf.LambdaFunctionNames, nil
}
