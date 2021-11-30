package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// AccessAnalyzer - represents all AWS secrets manager secrets that should be deleted.
type AccessAnalyzer struct {
	AnalyzerNames []string
}

// ResourceName - the simple name of the aws resource
func (analyzer AccessAnalyzer) ResourceName() string {
	return "accessanalyzer"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (analyzer AccessAnalyzer) ResourceIdentifiers() []string {
	return analyzer.AnalyzerNames
}

func (analyzer AccessAnalyzer) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that IAM Access Analyzer does not support bulk delete,
	// so we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

// Nuke - nuke 'em all!!!
func (analyzer AccessAnalyzer) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAccessAnalyzers(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
