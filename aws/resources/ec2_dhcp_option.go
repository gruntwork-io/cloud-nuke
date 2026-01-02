package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2DhcpOptionAPI defines the interface for EC2 DHCP Options operations.
type EC2DhcpOptionAPI interface {
	AssociateDhcpOptions(ctx context.Context, params *ec2.AssociateDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.AssociateDhcpOptionsOutput, error)
	DescribeDhcpOptions(ctx context.Context, params *ec2.DescribeDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteDhcpOptions(ctx context.Context, params *ec2.DeleteDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteDhcpOptionsOutput, error)
}

// NewEC2DhcpOptions creates a new EC2DhcpOption resource using the generic resource pattern.
func NewEC2DhcpOptions() AwsResource {
	return NewAwsResource(&resource.Resource[EC2DhcpOptionAPI]{
		ResourceTypeName: "ec2-dhcp-option",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2DhcpOptionAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2DedicatedHosts // Note: Using EC2DedicatedHosts as there's no dedicated config for DHCP options
		},
		Lister: listEC2DhcpOptions,
		Nuker: resource.MultiStepDeleter(
			disassociateDhcpOption,
			deleteDhcpOption,
		),
		PermissionVerifier: func(ctx context.Context, client EC2DhcpOptionAPI, id *string) error {
			_, err := client.DeleteDhcpOptions(ctx, &ec2.DeleteDhcpOptionsInput{
				DhcpOptionsId: id,
				DryRun:        aws.Bool(true),
			})
			return err
		},
	})
}

// listEC2DhcpOptions returns a list of DHCP option IDs that are eligible for nuking.
// It filters out DHCP options that are associated with default VPCs.
func listEC2DhcpOptions(ctx context.Context, client EC2DhcpOptionAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var dhcpOptionIds []*string

	paginator := ec2.NewDescribeDhcpOptionsPaginator(client, &ec2.DescribeDhcpOptionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, dhcpOption := range page.DhcpOptions {
			isEligibleForNuke := true

			// Check if the DHCP option is attached to any VPC
			// If the VPC is default, omit the DHCP option ID from results
			vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
				Filters: []types.Filter{
					{
						Name:   aws.String("dhcp-options-id"),
						Values: []string{aws.ToString(dhcpOption.DhcpOptionsId)},
					},
				},
			})
			if err != nil {
				logging.Debugf("[Failed] %s", err)
				continue
			}

			for _, vpc := range vpcs.Vpcs {
				// Skip if attached to default VPC
				if aws.ToBool(vpc.IsDefault) {
					logging.Debugf("[Skipping] %s is attached with a default vpc %s", aws.ToString(dhcpOption.DhcpOptionsId), aws.ToString(vpc.VpcId))
					isEligibleForNuke = false
				}
			}

			if isEligibleForNuke {
				dhcpOptionIds = append(dhcpOptionIds, dhcpOption.DhcpOptionsId)
			}
		}
	}

	return dhcpOptionIds, nil
}

// disassociateDhcpOption detaches the DHCP option from its associated VPCs.
func disassociateDhcpOption(ctx context.Context, client EC2DhcpOptionAPI, id *string) error {
	dhcpOptID := aws.ToString(id)
	logging.Debugf("[DHCP Option] Looking up VPC associations for: %s", dhcpOptID)

	// Query AWS to get the current VPC associations
	vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("dhcp-options-id"),
				Values: []string{dhcpOptID},
			},
		},
	})
	if err != nil {
		logging.Debugf("[DHCP Option] Failed to describe VPCs for %s: %s", dhcpOptID, err)
		return errors.WithStackTrace(err)
	}

	if len(vpcs.Vpcs) == 0 {
		logging.Debugf("[DHCP Option] No VPC associations found for %s, skipping disassociate", dhcpOptID)
		return nil
	}

	// Disassociate from all VPCs
	for _, vpc := range vpcs.Vpcs {
		vpcID := aws.ToString(vpc.VpcId)
		logging.Debugf("[DHCP Option] Disassociating %s from VPC %s", dhcpOptID, vpcID)

		_, err := client.AssociateDhcpOptions(ctx, &ec2.AssociateDhcpOptionsInput{
			VpcId:         vpc.VpcId,
			DhcpOptionsId: aws.String("default"), // "default" means no DHCP options
		})
		if err != nil {
			logging.Debugf("[DHCP Option] Failed to disassociate %s from VPC %s: %s", dhcpOptID, vpcID, err)
			return errors.WithStackTrace(err)
		}

		logging.Debugf("[DHCP Option] Successfully disassociated %s from VPC %s", dhcpOptID, vpcID)
	}

	return nil
}

// deleteDhcpOption deletes the DHCP option.
func deleteDhcpOption(ctx context.Context, client EC2DhcpOptionAPI, id *string) error {
	logging.Debugf("Deleting DHCP Option %s", aws.ToString(id))

	_, err := client.DeleteDhcpOptions(ctx, &ec2.DeleteDhcpOptionsInput{
		DhcpOptionsId: id,
	})
	if err != nil {
		logging.Debugf("[Failed] Error deleting DHCP option %s: %s", aws.ToString(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[Ok] DHCP Option deleted successfully %s", aws.ToString(id))
	return nil
}
