package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datapipeline"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

const describePipelinesBatchSize = 25

type DataPipelineAPI interface {
	ListPipelines(ctx context.Context, params *datapipeline.ListPipelinesInput, optFns ...func(*datapipeline.Options)) (*datapipeline.ListPipelinesOutput, error)
	DescribePipelines(ctx context.Context, params *datapipeline.DescribePipelinesInput, optFns ...func(*datapipeline.Options)) (*datapipeline.DescribePipelinesOutput, error)
	DeletePipeline(ctx context.Context, params *datapipeline.DeletePipelineInput, optFns ...func(*datapipeline.Options)) (*datapipeline.DeletePipelineOutput, error)
}

func NewDataPipeline() AwsResource {
	return NewAwsResource(&resource.Resource[DataPipelineAPI]{
		ResourceTypeName: "data-pipeline",
		BatchSize:        20,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DataPipelineAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = datapipeline.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DataPipeline
		},
		Lister: listDataPipelines,
		Nuker:  resource.SimpleBatchDeleter(deleteDataPipeline),
	})
}

func listDataPipelines(ctx context.Context, client DataPipelineAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := datapipeline.NewListPipelinesPaginator(client, &datapipeline.ListPipelinesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var pipelineIds []string
		for _, p := range page.PipelineIdList {
			pipelineIds = append(pipelineIds, aws.ToString(p.Id))
		}
		if len(pipelineIds) == 0 {
			continue
		}

		// DescribePipelines accepts max 25 IDs per call
		for i := 0; i < len(pipelineIds); i += describePipelinesBatchSize {
			end := min(i+describePipelinesBatchSize, len(pipelineIds))

			descOutput, err := client.DescribePipelines(ctx, &datapipeline.DescribePipelinesInput{
				PipelineIds: pipelineIds[i:end],
			})
			if err != nil {
				return nil, err
			}

			for _, desc := range descOutput.PipelineDescriptionList {
				pipelineID := aws.ToString(desc.PipelineId)
				name := aws.ToString(desc.Name)

				rv := config.ResourceValue{Name: &name}
				for _, field := range desc.Fields {
					if aws.ToString(field.Key) == "@creationTime" {
						rv.Time, _ = util.ParseTimestampPtr(field.StringValue)
						break
					}
				}

				if cfg.ShouldInclude(rv) {
					identifiers = append(identifiers, aws.String(pipelineID))
				}
			}
		}
	}

	return identifiers, nil
}

func deleteDataPipeline(ctx context.Context, client DataPipelineAPI, pipelineID *string) error {
	_, err := client.DeletePipeline(ctx, &datapipeline.DeletePipelineInput{
		PipelineId: pipelineID,
	})
	return err
}
