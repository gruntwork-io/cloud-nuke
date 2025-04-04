package resources

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

func (cd *CloudfrontDistribution) getAll(c context.Context, configObj config.Config) (ids []*string, err error) {
	var marker *string
	for {
		listOutput, err := cd.Client.ListDistributions(c, &cloudfront.ListDistributionsInput{
			Marker: marker,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if listOutput.DistributionList != nil && listOutput.DistributionList.Items != nil {
			for _, item := range listOutput.DistributionList.Items {
				if !configObj.CloudfrontDistribution.ShouldInclude(config.ResourceValue{
					Name: item.Id,
				}) {
					continue
				}

				ids = append(ids, item.Id)
			}
		}

		if listOutput.DistributionList.NextMarker == nil {
			break
		}
		marker = listOutput.DistributionList.NextMarker
	}

	return
}

func (cd *CloudfrontDistribution) nukeAll(ids []*string) (err error) {
	if len(ids) == 0 {
		logging.Debugf("No Cloudfront Distribution to nuke in region %s", cd.Region)
		return nil
	}

	logging.Debugf("Deleting all CCloudfront Distribution Trails in region %s", cd.Region)
	var deletedIds []*string
	ctx := context.Background()
	for _, id := range ids {
		getOutput, err := cd.Client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
			Id: id,
		})
		if err != nil {
			logging.Warnf("[Failed] to get distribution %s: %v\n", *id, err)
			continue
		}

		// needs to disable the distribution first
		if *getOutput.DistributionConfig.Enabled {
			logging.Debugf("Disabling distribution %s...\n", *id)

			getOutput.DistributionConfig.Enabled = aws.Bool(false)
			_, err = cd.Client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
				Id:                 id,
				IfMatch:            getOutput.ETag,
				DistributionConfig: getOutput.DistributionConfig,
			})
			if err != nil {
				logging.Warnf("[Failed] to disable distribution %s: %v\n", *id, err)
				continue
			}

			logging.Debugf("Waiting for distribution %s to be disabled...\n", *id)
			sleep := 2
			for {
				time.Sleep(2 * time.Second) // Wait before checking again
				getOutput, err = cd.Client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
					Id: id,
				})
				if err != nil {
					log.Printf("failed to get updated status for %s: %v\n", *id, err)
					continue
				}
				if !*getOutput.DistributionConfig.Enabled {
					break
				}
				if sleep > 32 {
					sleep = 32
				}
			}
		}
		_, err = cd.Client.DeleteDistribution(ctx, &cloudfront.DeleteDistributionInput{
			Id:      id,
			IfMatch: getOutput.ETag,
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Cloudfront Distribution: %s", *id)
		}
	}
	logging.Debugf("[OK] %d Cloudfront Distribution deleted in %s", len(deletedIds), cd.Region)
	return
}
