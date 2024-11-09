package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (h *EC2DedicatedHosts) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var hostIds []*string
	describeHostsInput := &ec2.DescribeHostsInput{
		Filter: []types.Filter{
			{
				Name: aws.String("state"),
				Values: []string{
					"available",
					"under-assessment",
					"permanent-failure",
				},
			},
		},
	}

	paginator := ec2.NewDescribeHostsPaginator(h.Client, describeHostsInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, host := range page.Hosts {
			if shouldIncludeHostId(&host, configObj) {
				hostIds = append(hostIds, host.HostId)
			}
		}
	}

	return hostIds, nil
}

func shouldIncludeHostId(host *types.Host, configObj config.Config) bool {
	if host == nil {
		return false
	}

	// If an instance is using the host allocation we cannot release it
	if len(host.Instances) != 0 {
		logging.Debugf("Host %s has instance(s) still associated, unable to nuke.", *host.HostId)
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	hostNameTagValue := util.GetEC2ResourceNameTagValue(host.Tags)

	return configObj.EC2DedicatedHosts.ShouldInclude(config.ResourceValue{
		Name: hostNameTagValue,
		Time: host.AllocationTime,
	})
}

func (h *EC2DedicatedHosts) nukeAll(hostIds []*string) error {
	if len(hostIds) == 0 {
		logging.Debugf("No EC2 dedicated hosts to nuke in region %s", h.Region)
		return nil
	}

	logging.Debugf("Releasing all EC2 dedicated host allocations in region %s", h.Region)

	input := &ec2.ReleaseHostsInput{HostIds: aws.ToStringSlice(hostIds)}

	releaseResult, err := h.Client.ReleaseHosts(h.Context, input)

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	// Report successes and failures from release host request
	for _, hostSuccess := range releaseResult.Successful {
		logging.Debugf("[OK] Dedicated host %s was released in %s", hostSuccess, h.Region)
		e := report.Entry{
			Identifier:   hostSuccess,
			ResourceType: "EC2 Dedicated Host",
		}
		report.Record(e)
	}

	for _, hostFailed := range releaseResult.Unsuccessful {
		logging.Debugf("[ERROR] Unable to release dedicated host %s in %s: %s", aws.ToString(hostFailed.ResourceId), h.Region, aws.ToString(hostFailed.Error.Message))
		e := report.Entry{
			Identifier:   aws.ToString(hostFailed.ResourceId),
			ResourceType: "EC2 Dedicated Host",
			Error:        fmt.Errorf(*hostFailed.Error.Message),
		}
		report.Record(e)
	}

	return nil
}
