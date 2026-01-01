package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
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
		Nuker:  resource.SimpleBatchDeleter(deleteLambdaLayerVersion),
	})
}

// listLambdaLayers retrieves all Lambda layers that match the config filters.
// Returns composite identifiers in the format "layerName:versionNumber" for each layer version.
func listLambdaLayers(ctx context.Context, client LambdaLayersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := lambda.NewListLayersPaginator(client, &lambda.ListLayersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, layer := range page.Layers {
			logging.Debugf("Found layer: %s", *layer.LayerName)

			if !shouldIncludeLambdaLayer(&layer, cfg) {
				continue
			}

			// List all versions for this layer
			versionsPaginator := lambda.NewListLayerVersionsPaginator(client, &lambda.ListLayerVersionsInput{
				LayerName: layer.LayerName,
			})
			for versionsPaginator.HasMorePages() {
				versionsPage, err := versionsPaginator.NextPage(ctx)
				if err != nil {
					return nil, errors.WithStackTrace(err)
				}

				for _, version := range versionsPage.LayerVersions {
					// Create composite identifier: layerName:versionNumber
					id := fmt.Sprintf("%s:%d", *layer.LayerName, version.Version)
					logging.Debugf("Found layer version: %s", id)
					identifiers = append(identifiers, aws.String(id))
				}
			}
		}
	}

	return identifiers, nil
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

// deleteLambdaLayerVersion deletes a single Lambda layer version.
// The id parameter is a composite identifier in the format "layerName:versionNumber".
func deleteLambdaLayerVersion(ctx context.Context, client LambdaLayersAPI, id *string) error {
	parts := strings.SplitN(aws.ToString(id), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid layer version identifier: %s", aws.ToString(id))
	}

	layerName := parts[0]
	versionNum, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid version number in identifier %s: %w", aws.ToString(id), err)
	}

	_, err = client.DeleteLayerVersion(ctx, &lambda.DeleteLayerVersionInput{
		LayerName:     aws.String(layerName),
		VersionNumber: aws.Int64(versionNum),
	})
	return err
}
