package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllEc2DedicatedHosts(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)
	var hostIds []*string

	describeHostsInput := &ec2.DescribeHostsInput{
		Filter: []*ec2.Filter{
			{
				Name: awsgo.String("state"),
				Values: []*string{
					awsgo.String("available"),
					awsgo.String("under-assessment"),
					awsgo.String("permanent-failure"),
				},
			},
		},
	}

	err := svc.DescribeHostsPages(
		describeHostsInput,
		func(page *ec2.DescribeHostsOutput, lastPage bool) bool {
			for _, host := range page.Hosts {
				if shouldIncludeHostId(host, excludeAfter, configObj) {
					hostIds = append(hostIds, host.HostId)
				}
			}
			return !lastPage
		},
	)

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return hostIds, nil
}

func shouldIncludeHostId(host *ec2.Host, excludeAfter time.Time, configObj config.Config) bool {
	if host == nil {
		return false
	}

	if excludeAfter.Before(*host.AllocationTime) {
		return false
	}

	// If an instance is using the host allocation we cannot release it
	if len(host.Instances) != 0 {
		logging.Logger.Debugf("Host %s has instance(s) still associated, unable to nuke.", *host.HostId)
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	hostNameTagValue, _ := GetEC2ResourceNameTagValue(host.Tags)

	return config.ShouldInclude(
		hostNameTagValue,
		configObj.EC2DedicatedHost.IncludeRule.NamesRegExp,
		configObj.EC2DedicatedHost.ExcludeRule.NamesRegExp,
	)
}

func nukeAllEc2DedicatedHosts(session *session.Session, hostIds []*string) error {
	svc := ec2.New(session)

	if len(hostIds) == 0 {
		logging.Logger.Debugf("No EC2Instance dedicated hosts to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Releasing all EC2Instance dedicated host allocations in region %s", *session.Config.Region)

	input := &ec2.ReleaseHostsInput{HostIds: hostIds}

	releaseResult, err := svc.ReleaseHosts(input)

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking EC2Instance Dedicated Hosts",
		}, map[string]interface{}{
			"region": *session.Config.Region,
		})
		return errors.WithStackTrace(err)
	}

	// Report successes and failures from release host request
	for _, hostSuccess := range releaseResult.Successful {
		logging.Logger.Debugf("[OK] Dedicated host %s was released in %s", aws.StringValue(hostSuccess), *session.Config.Region)
		e := report.Entry{
			Identifier:   aws.StringValue(hostSuccess),
			ResourceType: "EC2Instance Dedicated Host",
		}
		report.Record(e)
	}

	for _, hostFailed := range releaseResult.Unsuccessful {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking EC2Instance Dedicated Host",
		}, map[string]interface{}{
			"region": *session.Config.Region,
		})
		logging.Logger.Debugf("[ERROR] Unable to release dedicated host %s in %s: %s", aws.StringValue(hostFailed.ResourceId), *session.Config.Region, aws.StringValue(hostFailed.Error.Message))
		e := report.Entry{
			Identifier:   aws.StringValue(hostFailed.ResourceId),
			ResourceType: "EC2Instance Dedicated Host",
			Error:        fmt.Errorf(*hostFailed.Error.Message),
		}
		report.Record(e)
	}

	return nil
}
