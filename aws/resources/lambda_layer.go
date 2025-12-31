package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// LambdaLayersAPI defines the interface for Lambda layer operations.
type LambdaLayersAPI interface {
	DeleteLayerVersion(ctx context.Context, params *lambda.DeleteLayerVersionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteLayerVersionOutput, error)
	ListLayers(ctx context.Context, params *lambda.ListLayersInput, optFns ...func(*lambda.Options)) (*lambda.ListLayersOutput, error)
	ListLayerVersions(ctx context.Context, params *lambda.ListLayerVersionsInput, optFns ...func(*lambda.Options)) (*lambda.ListLayerVersionsOutput, error)
}

// NewLambdaLayers creates a new LambdaLayers resource using the generic resource pattern.
func NewLambdaLayers() AwsResource {
	return NewAwsResource(&resource.Resource[LambdaLayersAPI]{
		ResourceTypeName: "lambda_layer",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[LambdaLayersAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for LambdaLayers client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = lambda.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.LambdaLayer
		},
		Lister: listLambdaLayers,
		Nuker:  deleteLambdaLayers,
	})
}

// listLambdaLayers retrieves all Lambda layers that match the config filters.
func listLambdaLayers(ctx context.Context, client LambdaLayersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var layers []types.LayersListItem
	var names []*string

	paginator := lambda.NewListLayersPaginator(client, &lambda.ListLayersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, layer := range page.Layers {
			logging.Debugf("Found layer! %s", *layer.LayerName)

			if shouldIncludeLambdaLayer(&layer, cfg) {
				layers = append(layers, layer)
			}
		}
	}

	for _, layer := range layers {
		versionsPaginator := lambda.NewListLayerVersionsPaginator(client, &lambda.ListLayerVersionsInput{
			LayerName: layer.LayerName,
		})
		for versionsPaginator.HasMorePages() {
			page, err := versionsPaginator.NextPage(ctx)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			for _, version := range page.LayerVersions {
				logging.Debugf("Found layer version! %d", version.Version)

				// Currently the output is just the identifier which is the layer's name.
				// There could be potentially multiple rows of the same identifier or
				// layer name since there can be multiple versions of it.
				names = append(names, layer.LayerName)
			}
		}
	}

	return names, nil
}

func shouldIncludeLambdaLayer(lambdaLayer *types.LayersListItem, cfg config.ResourceType) bool {
	if lambdaLayer == nil {
		return false
	}

	// Lambda layers are immutable, so the created date of the latest version
	// is on par with last modified
	fnLastModified := aws.ToString(lambdaLayer.LatestMatchingVersion.CreatedDate)
	fnName := lambdaLayer.LayerName
	layout := "2006-01-02T15:04:05.000+0000"
	lastModifiedDateTime, err := time.Parse(layout, fnLastModified)
	if err != nil {
		logging.Debugf("Could not parse last modified timestamp (%s) of Lambda layer %s. Excluding from delete.", fnLastModified, *fnName)
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Time: &lastModifiedDateTime,
		Name: fnName,
	})
}

// deleteLambdaLayers is a custom nuker function for Lambda layers.
// It needs to delete all versions of each layer, which requires special handling.
func deleteLambdaLayers(ctx context.Context, client LambdaLayersAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Lambda Layers to nuke in %s", scope)
		return nil
	}

	logging.Infof("Deleting %d Lambda Layers in %s", len(identifiers), scope)
	var deletedNames []*string
	var deleteLayerVersions []*lambda.DeleteLayerVersionInput

	for _, name := range identifiers {
		paginator := lambda.NewListLayerVersionsPaginator(client, &lambda.ListLayerVersionsInput{
			LayerName: name,
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return errors.WithStackTrace(err)
			}

			for _, version := range page.LayerVersions {
				logging.Debugf("Found layer version! %s", *version.LayerVersionArn)
				params := &lambda.DeleteLayerVersionInput{
					LayerName:     name,
					VersionNumber: aws.Int64(version.Version),
				}
				deleteLayerVersions = append(deleteLayerVersions, params)
			}
		}
	}

	for _, params := range deleteLayerVersions {

		_, err := client.DeleteLayerVersion(ctx, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(params.LayerName),
			ResourceType: resourceType,
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *params.LayerName, err)
		} else {
			deletedNames = append(deletedNames, params.LayerName)
			logging.Debugf("Deleted Lambda Layer: %s", aws.ToString(params.LayerName))
		}
	}

	logging.Debugf("[OK] %d Lambda Layer(s) deleted in %s", len(deletedNames), scope)
	return nil
}

type LambdaVersionDeleteError struct {
	name string
}

func (e LambdaVersionDeleteError) Error() string {
	return "Lambda Function:" + e.name + "was not deleted"
}
