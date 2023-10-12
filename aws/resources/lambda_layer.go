package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (ll *LambdaLayers) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var layers []*lambda.LayersListItem
	var names []*string

	err := ll.Client.ListLayersPages(
		&lambda.ListLayersInput{}, func(page *lambda.ListLayersOutput, lastPage bool) bool {
			for _, layer := range page.Layers {
				logging.Logger.Debugf("Found layer! %s", layer)

				if ll.shouldInclude(layer, configObj) {
					layers = append(layers, layer)
				}
			}

			return !lastPage
		})

	if err != nil {
		return nil, err
	}

	for _, layer := range layers {
		err := ll.Client.ListLayerVersionsPages(
			&lambda.ListLayerVersionsInput{
				LayerName: layer.LayerName,
			}, func(page *lambda.ListLayerVersionsOutput, lastPage bool) bool {
				for _, version := range page.LayerVersions {
					logging.Logger.Debugf("Found layer version! %s", version)
					names = append(names, layer.LayerName)
				}

				return !lastPage
			})

		if err != nil {
			return nil, err
		}
	}

	return names, nil
}

func (ll *LambdaLayers) shouldInclude(lambdaLayer *lambda.LayersListItem, configObj config.Config) bool {
	if lambdaLayer == nil {
		return false
	}

	// fnLastModified := aws.StringValue(lambdaLayer.LastModified)
	fnLastModified := aws.StringValue(lambdaLayer.LatestMatchingVersion.CreatedDate)
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
	deletedNames := []*string{}
	deleteLayerVersions := []*lambda.DeleteLayerVersionInput{}

	for _, name := range names {
		err := ll.Client.ListLayerVersionsPages(
			&lambda.ListLayerVersionsInput{
				LayerName: name,
			}, func(page *lambda.ListLayerVersionsOutput, lastPage bool) bool {
				for _, version := range page.LayerVersions {
					logging.Logger.Infof("Found layer version! %s", version)
					params := &lambda.DeleteLayerVersionInput{
						LayerName:     name,
						VersionNumber: version.Version,
					}
					deleteLayerVersions = append(deleteLayerVersions, params)
				}

				return !lastPage
			})

		if err != nil {
			return err
		}
	}

	for _, params := range deleteLayerVersions {

		_, err := ll.Client.DeleteLayerVersion(params)

		if err != nil {
			return err
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(params.LayerName),
			ResourceType: "Lambda layer",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *params.LayerName, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Lambda Layer",
			}, map[string]interface{}{
				"region": ll.Region,
			})
		} else {
			deletedNames = append(deletedNames, params.LayerName)
			logging.Logger.Debugf("Deleted Lambda Layer: %s", awsgo.StringValue(params.LayerName))
		}
	}

	logging.Logger.Debugf("[OK] %d Lambda Function(s) deleted in %s", len(deletedNames), ll.Region)
	return nil
}
