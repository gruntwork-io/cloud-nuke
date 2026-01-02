package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EC2IPAMByoasnAPI defines the interface for EC2 IPAM BYOASN operations.
type EC2IPAMByoasnAPI interface {
	DescribeIpamByoasn(ctx context.Context, params *ec2.DescribeIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamByoasnOutput, error)
	DisassociateIpamByoasn(ctx context.Context, params *ec2.DisassociateIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateIpamByoasnOutput, error)
}

// NewEC2IPAMByoasn creates a new EC2 IPAM BYOASN resource using the generic resource pattern.
func NewEC2IPAMByoasn() AwsResource {
	return NewAwsResource(&resource.Resource[EC2IPAMByoasnAPI]{
		ResourceTypeName: "ipam-byoasn",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMByoasnAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAMByoasn
		},
		Lister:             listEC2IPAMByoasns,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2IPAMByoasn),
		PermissionVerifier: verifyEC2IPAMByoasnPermission,
	})
}

// listEC2IPAMByoasns retrieves all IPAM BYOASNs.
// Note: DescribeIpamByoasn does not support pagination.
func listEC2IPAMByoasns(ctx context.Context, client EC2IPAMByoasnAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	output, err := client.DescribeIpamByoasn(ctx, &ec2.DescribeIpamByoasnInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return nil, err
	}

	for _, byoasn := range output.Byoasns {
		result = append(result, byoasn.Asn)
	}

	return result, nil
}

// verifyEC2IPAMByoasnPermission performs a dry-run disassociate to check permissions.
func verifyEC2IPAMByoasnPermission(ctx context.Context, client EC2IPAMByoasnAPI, id *string) error {
	_, err := client.DisassociateIpamByoasn(ctx, &ec2.DisassociateIpamByoasnInput{
		Asn:    id,
		DryRun: aws.Bool(true),
	})
	return err
}

// deleteEC2IPAMByoasn disassociates a single IPAM BYOASN.
func deleteEC2IPAMByoasn(ctx context.Context, client EC2IPAMByoasnAPI, id *string) error {
	_, err := client.DisassociateIpamByoasn(ctx, &ec2.DisassociateIpamByoasnInput{
		Asn: id,
	})
	return err
}
