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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2DedicatedHostsAPI defines the interface for EC2 Dedicated Hosts operations.
type EC2DedicatedHostsAPI interface {
	DescribeHosts(ctx context.Context, params *ec2.DescribeHostsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeHostsOutput, error)
	ReleaseHosts(ctx context.Context, params *ec2.ReleaseHostsInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseHostsOutput, error)
}

// NewEC2DedicatedHosts creates a new EC2DedicatedHosts resource using the generic resource pattern.
func NewEC2DedicatedHosts() AwsResource {
	return NewAwsResource(&resource.Resource[EC2DedicatedHostsAPI]{
		ResourceTypeName: "ec2-dedicated-hosts",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[EC2DedicatedHostsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2DedicatedHosts
		},
		Lister: listEC2DedicatedHosts,
		Nuker:  deleteEC2DedicatedHosts,
	})
}

// listEC2DedicatedHosts retrieves all EC2 dedicated hosts that match the config filters.
func listEC2DedicatedHosts(ctx context.Context, client EC2DedicatedHostsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
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

	paginator := ec2.NewDescribeHostsPaginator(client, describeHostsInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, host := range page.Hosts {
			if shouldIncludeHostId(&host, cfg) {
				hostIds = append(hostIds, host.HostId)
			}
		}
	}

	return hostIds, nil
}

// shouldIncludeHostId determines if an EC2 dedicated host should be included for deletion.
func shouldIncludeHostId(host *types.Host, cfg config.ResourceType) bool {
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

	return cfg.ShouldInclude(config.ResourceValue{
		Name: hostNameTagValue,
		Time: host.AllocationTime,
	})
}

// deleteEC2DedicatedHosts releases all EC2 dedicated hosts.
func deleteEC2DedicatedHosts(ctx context.Context, client EC2DedicatedHostsAPI, scope resource.Scope, resourceType string, hostIds []*string) error {
	if len(hostIds) == 0 {
		logging.Debugf("No EC2 dedicated hosts to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Releasing all EC2 dedicated host allocations in region %s", scope.Region)

	input := &ec2.ReleaseHostsInput{HostIds: aws.ToStringSlice(hostIds)}

	releaseResult, err := client.ReleaseHosts(ctx, input)

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	// Report successes and failures from release host request
	for _, hostSuccess := range releaseResult.Successful {
		logging.Debugf("[OK] Dedicated host %s was released in %s", hostSuccess, scope.Region)
		e := report.Entry{
			Identifier:   hostSuccess,
			ResourceType: "EC2 Dedicated Host",
		}
		report.Record(e)
	}

	for _, hostFailed := range releaseResult.Unsuccessful {
		logging.Debugf("[ERROR] Unable to release dedicated host %s in %s: %s", aws.ToString(hostFailed.ResourceId), scope.Region, aws.ToString(hostFailed.Error.Message))
		e := report.Entry{
			Identifier:   aws.ToString(hostFailed.ResourceId),
			ResourceType: "EC2 Dedicated Host",
			Error:        fmt.Errorf("%s", *hostFailed.Error.Message),
		}
		report.Record(e)
	}

	return nil
}
