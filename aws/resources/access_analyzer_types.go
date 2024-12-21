package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type AccessAnalyzerAPI interface {
	DeleteAnalyzer(ctx context.Context, params *accessanalyzer.DeleteAnalyzerInput, optFns ...func(*accessanalyzer.Options)) (*accessanalyzer.DeleteAnalyzerOutput, error)
	ListAnalyzers(context.Context, *accessanalyzer.ListAnalyzersInput, ...func(*accessanalyzer.Options)) (*accessanalyzer.ListAnalyzersOutput, error)
}

// AccessAnalyzer - represents all AWS secrets manager secrets that should be deleted.
type AccessAnalyzer struct {
	BaseAwsResource
	Client        AccessAnalyzerAPI
	Region        string
	AnalyzerNames []string
}

func (analyzer *AccessAnalyzer) Init(cfg aws.Config) {
	analyzer.Client = accessanalyzer.NewFromConfig(cfg)
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

// GetAndSetResourceConfig To get the resource configuration
func (analyzer *AccessAnalyzer) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.AccessAnalyzer
}

func (analyzer *AccessAnalyzer) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := analyzer.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	analyzer.AnalyzerNames = aws.ToStringSlice(identifiers)
	return analyzer.AnalyzerNames, nil
}

// Nuke - nuke 'em all!!!
func (analyzer *AccessAnalyzer) Nuke(identifiers []string) error {
	if err := analyzer.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
