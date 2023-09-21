package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/accessanalyzer"
	"github.com/aws/aws-sdk-go/service/accessanalyzer/accessanalyzeriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// AccessAnalyzer - represents all AWS secrets manager secrets that should be deleted.
type AccessAnalyzer struct {
	Client        accessanalyzeriface.AccessAnalyzerAPI
	Region        string
	AnalyzerNames []string
}

func (analyzer *AccessAnalyzer) Init(session *session.Session) {
	analyzer.Client = accessanalyzer.New(session)
}

// ResourceName - the simple name of the aws resource
func (analyzer *AccessAnalyzer) ResourceName() string {
	return "accessanalyzer"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (analyzer *AccessAnalyzer) ResourceIdentifiers() []string {
	return analyzer.AnalyzerNames
}

func (analyzer *AccessAnalyzer) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that IAM Access Analyzer does not support bulk delete,
	// so we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (analyzer *AccessAnalyzer) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := analyzer.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	analyzer.AnalyzerNames = awsgo.StringValueSlice(identifiers)
	return analyzer.AnalyzerNames, nil
}

// Nuke - nuke 'em all!!!
func (analyzer *AccessAnalyzer) Nuke(identifiers []string) error {
	if err := analyzer.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
