package resources

import (
	"github.com/gruntwork-io/cloud-nuke/util"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (v *EC2VPCs) setFirstSeenTag(vpc ec2.Vpc, value time.Time) error {
	// We set a first seen tag because an Elastic IP doesn't contain an attribute that gives us it's creation time
	_, err := v.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{vpc.VpcId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestampTag(value)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (v *EC2VPCs) getFirstSeenTag(vpc ec2.Vpc) (*time.Time, error) {
	tags := vpc.Tags
	for _, tag := range tags {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestampTag(tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return nil, nil
}

func (v *EC2VPCs) getAll(configObj config.Config) ([]*string, error) {
	result, err := v.Client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			// Note: this filter omits the default since there is special
			// handling for default resources already
			{
				Name:   awsgo.String("is-default"),
				Values: awsgo.StringSlice([]string{"false"}),
			},
		},
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, vpc := range result.Vpcs {
		firstSeenTime, err := v.getFirstSeenTag(*vpc)
		if err != nil {
			logging.Logger.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := v.setFirstSeenTag(*vpc, time.Now().UTC()); err != nil {
				return nil, err
			}
		}

		if configObj.VPC.ShouldInclude(config.ResourceValue{
			Time: firstSeenTime,
			Name: GetEC2ResourceNameTagValue(vpc.Tags),
		}) {
			ids = append(ids, vpc.VpcId)
		}
	}

	return ids, nil
}

func (v *EC2VPCs) nukeAll(vpcIds []string) error {
	if len(vpcIds) == 0 {
		logging.Logger.Debug("No VPCs to nuke")
		return nil
	}

	logging.Logger.Debug("Deleting all VPCs")

	deletedVPCs := 0
	multiErr := new(multierror.Error)

	for _, id := range vpcIds {
		_, err := v.Client.DeleteVpc(&ec2.DeleteVpcInput{
			VpcId: awsgo.String(id),
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   id,
			ResourceType: "VPC",
			Error:        err,
		}
		report.Record(e)

		if err != nil {

			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking VPC",
			}, map[string]interface{}{
				"region": v.Region,
			})
			multierror.Append(multiErr, err)
		} else {
			deletedVPCs++
			logging.Logger.Debugf("Deleted VPC: %s", id)
		}
	}

	logging.Logger.Debugf("[OK] %d VPC terminated", deletedVPCs)

	return nil
}
