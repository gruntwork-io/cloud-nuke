package resources

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2VpcAPI is the interface for the EC2 VPC client.
type EC2VpcAPI interface {
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// ec2VpcResource holds extra state needed for EC2 VPC operations.
type ec2VpcResource struct {
	*resource.Resource[EC2VpcAPI]
	defaultOnly bool
}

// NewEC2VPC creates a new EC2 VPC resource using the generic resource pattern.
func NewEC2VPC() AwsResource {
	r := &ec2VpcResource{
		Resource: &resource.Resource[EC2VpcAPI]{
			ResourceTypeName: "vpc",
			BatchSize:        49,
		},
	}

	r.InitClient = WrapAwsInitClient(func(res *resource.Resource[EC2VpcAPI], cfg aws.Config) {
		res.Scope.Region = cfg.Region
		res.Client = ec2.NewFromConfig(cfg)
	})

	r.ConfigGetter = func(c config.Config) config.ResourceType {
		// Store the DefaultOnly flag for use in the lister
		r.defaultOnly = c.VPC.DefaultOnly
		return c.VPC.ResourceType
	}

	r.Lister = func(ctx context.Context, client EC2VpcAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
		return listVPCs(ctx, client, scope, cfg, r.defaultOnly)
	}

	r.Nuker = resource.SequentialDeleter(deleteVPC)

	return &AwsResourceAdapter[EC2VpcAPI]{Resource: r.Resource}
}

func listVPCs(ctx context.Context, client EC2VpcAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
	var ids []*string
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{strconv.FormatBool(defaultOnly)},
			},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, vpc := range page.Vpcs {
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, vpc.VpcId, util.ConvertTypesTagsToMap(vpc.Tags))
			if err != nil {
				logging.Errorf("Unable to retrieve first seen tag for VPC %s: %v", aws.ToString(vpc.VpcId), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: firstSeenTime,
				Name: util.GetEC2ResourceNameTagValue(vpc.Tags),
				Tags: util.ConvertTypesTagsToMap(vpc.Tags),
			}) {
				ids = append(ids, vpc.VpcId)
			}
		}
	}

	return ids, nil
}

func deleteVPC(ctx context.Context, client EC2VpcAPI, id *string) error {
	logging.Debugf("Deleting VPC %s", aws.ToString(id))

	if _, err := client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: id,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Successfully deleted VPC %s", aws.ToString(id))
	return nil
}
