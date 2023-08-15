package resources

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (h *EC2DedicatedHosts) getAll(configObj config.Config) ([]*string, error) {
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

	err := h.Client.DescribeHostsPages(
		describeHostsInput,
		func(page *ec2.DescribeHostsOutput, lastPage bool) bool {
			for _, host := range page.Hosts {
				if shouldIncludeHostId(host, configObj) {
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

func shouldIncludeHostId(host *ec2.Host, configObj config.Config) bool {
	if host == nil {
		return false
	}

	// If an instance is using the host allocation we cannot release it
	if len(host.Instances) != 0 {
		logging.Logger.Debugf("Host %s has instance(s) still associated, unable to nuke.", *host.HostId)
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	hostNameTagValue := GetEC2ResourceNameTagValue(host.Tags)

	return configObj.EC2DedicatedHosts.ShouldInclude(config.ResourceValue{
		Name: hostNameTagValue,
		Time: host.AllocationTime,
	})
}

func (h *EC2DedicatedHosts) nukeAll(hostIds []*string) error {
	if len(hostIds) == 0 {
		logging.Logger.Debugf("No EC2 dedicated hosts to nuke in region %s", h.Region)
		return nil
	}

	logging.Logger.Debugf("Releasing all EC2 dedicated host allocations in region %s", h.Region)

	input := &ec2.ReleaseHostsInput{HostIds: hostIds}

	releaseResult, err := h.Client.ReleaseHosts(input)

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking EC2 Dedicated Hosts",
		}, map[string]interface{}{
			"region": h.Region,
		})
		return errors.WithStackTrace(err)
	}

	// Report successes and failures from release host request
	for _, hostSuccess := range releaseResult.Successful {
		logging.Logger.Debugf("[OK] Dedicated host %s was released in %s", aws.StringValue(hostSuccess), h.Region)
		e := report.Entry{
			Identifier:   aws.StringValue(hostSuccess),
			ResourceType: "EC2 Dedicated Host",
		}
		report.Record(e)
	}

	for _, hostFailed := range releaseResult.Unsuccessful {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking EC2 Dedicated Host",
		}, map[string]interface{}{
			"region": h.Region,
		})
		logging.Logger.Debugf("[ERROR] Unable to release dedicated host %s in %s: %s", aws.StringValue(hostFailed.ResourceId), h.Region, aws.StringValue(hostFailed.Error.Message))
		e := report.Entry{
			Identifier:   aws.StringValue(hostFailed.ResourceId),
			ResourceType: "EC2 Dedicated Host",
			Error:        fmt.Errorf(*hostFailed.Error.Message),
		}
		report.Record(e)
	}

	return nil
}
