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
	"github.com/gruntwork-io/go-commons/errors"
)

func (ll *LambdaLayers) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var layers []types.LayersListItem
	var names []*string

	paginator := lambda.NewListLayersPaginator(ll.Client, &lambda.ListLayersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, layer := range page.Layers {
			logging.Logger.Debugf("Found layer! %s", *layer.LayerName)

			if ll.shouldInclude(&layer, configObj) {
				layers = append(layers, layer)
			}
		}
	}

	for _, layer := range layers {

		versionsPaginator := lambda.NewListLayerVersionsPaginator(ll.Client, &lambda.ListLayerVersionsInput{
			LayerName: layer.LayerName,
		})
		for versionsPaginator.HasMorePages() {
			page, err := versionsPaginator.NextPage(c)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			for _, version := range page.LayerVersions {
				logging.Logger.Debugf("Found layer version! %d", version.Version)

				// Currently the output is just the identifier which is the layer's name.
				// There could be potentially multiple rows of the same identifier or
				// layer name since there can be multiple versions of it.
				names = append(names, layer.LayerName)
			}
		}
	}

	return names, nil
}

func (ll *LambdaLayers) shouldInclude(lambdaLayer *types.LayersListItem, configObj config.Config) bool {
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
		logging.Logger.Debugf("Could not parse last modified timestamp (%s) of Lambda layer %s. Excluding from delete.", fnLastModified, *fnName)
		return false
	}

	return configObj.LambdaLayer.ShouldInclude(config.ResourceValue{
		Time: &lastModifiedDateTime,
		Name: fnName,
	})
}

func (ll *LambdaLayers) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Logger.Debugf("No Lambda Layers to nuke in region %s", ll.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Lambda Layers in region %s", ll.Region)
	var deletedNames []*string
	var deleteLayerVersions []*lambda.DeleteLayerVersionInput

	for _, name := range names {
		paginator := lambda.NewListLayerVersionsPaginator(ll.Client, &lambda.ListLayerVersionsInput{
			LayerName: name,
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ll.Context)
			if err != nil {
				return errors.WithStackTrace(err)
			}

			for _, version := range page.LayerVersions {
				logging.Logger.Debugf("Found layer version! %s", *version.LayerVersionArn)
				params := &lambda.DeleteLayerVersionInput{
					LayerName:     name,
					VersionNumber: aws.Int64(version.Version),
				}
				deleteLayerVersions = append(deleteLayerVersions, params)
			}
		}
	}

	for _, params := range deleteLayerVersions {

		_, err := ll.Client.DeleteLayerVersion(ll.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(params.LayerName),
			ResourceType: "Lambda layer",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *params.LayerName, err)
		} else {
			deletedNames = append(deletedNames, params.LayerName)
			logging.Logger.Debugf("Deleted Lambda Layer: %s", aws.ToString(params.LayerName))
		}
	}

	logging.Logger.Debugf("[OK] %d Lambda Layer(s) deleted in %s", len(deletedNames), ll.Region)
	return nil
}
