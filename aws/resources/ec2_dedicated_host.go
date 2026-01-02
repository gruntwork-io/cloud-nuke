package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
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
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2DedicatedHostsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2DedicatedHosts
		},
		Lister: listEC2DedicatedHosts,
		Nuker:  resource.BulkResultDeleter(releaseEC2DedicatedHosts),
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

// releaseEC2DedicatedHosts releases EC2 dedicated hosts and returns per-item results.
func releaseEC2DedicatedHosts(ctx context.Context, client EC2DedicatedHostsAPI, hostIds []string) []resource.NukeResult {
	input := &ec2.ReleaseHostsInput{HostIds: hostIds}
	releaseResult, err := client.ReleaseHosts(ctx, input)
	if err != nil {
		releaseErr := errors.WithStackTrace(err)
		results := make([]resource.NukeResult, len(hostIds))
		for i, id := range hostIds {
			results[i] = resource.NukeResult{Identifier: id, Error: releaseErr}
		}
		return results
	}

	var results []resource.NukeResult
	for _, id := range releaseResult.Successful {
		results = append(results, resource.NukeResult{Identifier: id, Error: nil})
	}
	for _, item := range releaseResult.Unsuccessful {
		results = append(results, resource.NukeResult{
			Identifier: aws.ToString(item.ResourceId),
			Error:      fmt.Errorf("%s", aws.ToString(item.Error.Message)),
		})
	}

	return results
}
