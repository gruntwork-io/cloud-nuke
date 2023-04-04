package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/pterm/pterm"
)

func setFirstSeenVpcTag(svc *ec2.EC2, vpc ec2.Vpc, key string, value time.Time) error {
	// We set a first seen tag because an Elastic IP doesn't contain an attribute that gives us it's creation time
	_, err := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{vpc.VpcId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(key),
				Value: awsgo.String(value.Format(time.RFC3339)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func getFirstSeenVpcTag(vpc ec2.Vpc, key string) (*time.Time, error) {
	tags := vpc.Tags
	for _, tag := range tags {
		if *tag.Key == key {
			firstSeenTime, err := time.Parse(time.RFC3339, *tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return &firstSeenTime, nil
		}
	}

	return nil, nil
}

func getAllVpcs(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, []Vpc, error) {
	svc := ec2.New(session)

	result, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{
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
		return nil, nil, errors.WithStackTrace(err)
	}

	var ids []*string
	var vpcs []Vpc
	for _, vpc := range result.Vpcs {
		firstSeenTime, err := getFirstSeenVpcTag(*vpc, firstSeenTagKey)
		if err != nil {
			logging.Logger.Error("Unable to retrieve tags")
			return nil, nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := setFirstSeenVpcTag(svc, *vpc, firstSeenTagKey, time.Now().UTC()); err != nil {
				return nil, nil, err
			}
		}

		if shouldIncludeVpc(vpc, excludeAfter, *firstSeenTime, configObj) {
			ids = append(ids, vpc.VpcId)

			vpcs = append(vpcs, Vpc{
				VpcId:  *vpc.VpcId,
				Region: region,
				svc:    svc,
			})
		}
	}

	return ids, vpcs, nil
}

func shouldIncludeVpc(vpc *ec2.Vpc, excludeAfter time.Time, firstSeenTime time.Time, configObj config.Config) bool {
	if vpc == nil {
		return false
	}

	if excludeAfter.Before(firstSeenTime) {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	vpcName, _ := GetEC2ResourceNameTagValue(vpc.Tags)

	return config.ShouldInclude(
		vpcName,
		configObj.VPC.IncludeRule.NamesRegExp,
		configObj.VPC.ExcludeRule.NamesRegExp,
	)
}

func nukeAllVPCs(session *session.Session, vpcIds []string, vpcs []Vpc) error {
	if len(vpcIds) == 0 {
		logging.Logger.Debug("No VPCs to nuke")
		return nil
	}

	spinnerMsg := fmt.Sprintf("Deleting the following VPCs: %+v\n", vpcIds)

	// Start a simple spinner to track progress reading all relevant AWS resources
	spinnerSuccess, spinnerErr := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(spinnerMsg)

	if spinnerErr != nil {
		return errors.WithStackTrace(spinnerErr)
	}

	logging.Logger.Debug("Deleting all VPCs")

	deletedVPCs := 0
	multiErr := new(multierror.Error)

	for _, vpc := range vpcs {
		err := vpc.nuke(spinnerSuccess)
		// Record status of this resource
		e := report.Entry{
			Identifier:   vpc.VpcId,
			ResourceType: "VPC",
			Error:        err,
		}
		report.Record(e)

		if err != nil {

			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking VPC",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			multierror.Append(multiErr, err)
		} else {
			deletedVPCs++
			logging.Logger.Debugf("Deleted VPC: %s", vpc.VpcId)
		}
	}

	logging.Logger.Debugf("[OK] %d VPC terminated", deletedVPCs)

	return nil
}
