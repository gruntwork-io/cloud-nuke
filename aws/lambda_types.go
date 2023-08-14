package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LambdaFunctions struct {
	Client              lambdaiface.LambdaAPI
	Region              string
	LambdaFunctionNames []string
}

func (lf *LambdaFunctions) Init(session *session.Session) {
	lf.Client = lambda.New(session)
}

func (lf *LambdaFunctions) ResourceName() string {
	return "lambda"
}

// ResourceIdentifiers - The names of the lambda functions
func (lf *LambdaFunctions) ResourceIdentifiers() []string {
	return lf.LambdaFunctionNames
}

func (lf *LambdaFunctions) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (lf *LambdaFunctions) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := lf.getAll(configObj)
	if err != nil {
		return nil, err
	}

	lf.LambdaFunctionNames = awsgo.StringValueSlice(identifiers)
	return lf.LambdaFunctionNames, nil
}

// Nuke - nuke 'em all!!!
func (lf *LambdaFunctions) Nuke(identifiers []string) error {
	if err := lf.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type LambdaDeleteError struct {
	name string
}

func (e LambdaDeleteError) Error() string {
	return "Lambda Function:" + e.name + "was not deleted"
}
