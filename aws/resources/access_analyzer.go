package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AccessAnalyzerAPI defines the interface for AccessAnalyzer operations.
type AccessAnalyzerAPI interface {
	ListAnalyzers(ctx context.Context, params *accessanalyzer.ListAnalyzersInput, optFns ...func(*accessanalyzer.Options)) (*accessanalyzer.ListAnalyzersOutput, error)
	DeleteAnalyzer(ctx context.Context, params *accessanalyzer.DeleteAnalyzerInput, optFns ...func(*accessanalyzer.Options)) (*accessanalyzer.DeleteAnalyzerOutput, error)
}

// NewAccessAnalyzer creates a new AccessAnalyzer resource using the generic resource pattern.
func NewAccessAnalyzer() AwsResource {
	return NewAwsResource(&resource.Resource[AccessAnalyzerAPI]{
		ResourceTypeName: "accessanalyzer",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[AccessAnalyzerAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for AccessAnalyzer client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = accessanalyzer.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AccessAnalyzer
		},
		Lister: listAccessAnalyzers,
		Nuker:  resource.SimpleBatchDeleter(deleteAccessAnalyzer),
	})
}

// listAccessAnalyzers retrieves all IAM Access Analyzers that match the config filters.
func listAccessAnalyzers(ctx context.Context, client AccessAnalyzerAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allAnalyzers []*string
	paginator := accessanalyzer.NewListAnalyzersPaginator(client, &accessanalyzer.ListAnalyzersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, analyzer := range page.Analyzers {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: analyzer.CreatedAt,
				Name: analyzer.Name,
			}) {
				allAnalyzers = append(allAnalyzers, analyzer.Name)
			}
		}
	}

	return allAnalyzers, nil
}

// deleteAccessAnalyzer deletes a single IAM Access Analyzer.
func deleteAccessAnalyzer(ctx context.Context, client AccessAnalyzerAPI, analyzerName *string) error {
	_, err := client.DeleteAnalyzer(ctx, &accessanalyzer.DeleteAnalyzerInput{
		AnalyzerName: analyzerName,
	})
	return err
}
