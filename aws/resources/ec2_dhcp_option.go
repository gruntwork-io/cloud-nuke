package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
)

func (v *EC2DhcpOption) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	var dhcpOptionIds []*string

	paginator := ec2.NewDescribeDhcpOptionsPaginator(v.Client, &ec2.DescribeDhcpOptionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, dhcpOption := range page.DhcpOptions {
			var isEligibleForNuke = true

			// check the dhcp is attached with any vpc
			// if the vpc is default one, then omit the dhcp option id from result
			vpcs, err := v.Client.DescribeVpcs(v.Context, &ec2.DescribeVpcsInput{
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
				// check the vpc is the default one then set isEligible false as we dont need to remove the dhcp option of default vpc
				if aws.ToBool(vpc.IsDefault) {
					logging.Debugf("[Skipping] %s is attached with a default vpc %s", aws.ToString(dhcpOption.DhcpOptionsId), aws.ToString(vpc.VpcId))
					isEligibleForNuke = false
				}

				v.DhcpOptions[aws.ToString(dhcpOption.DhcpOptionsId)] = DHCPOption{
					Id:    dhcpOption.DhcpOptionsId,
					VpcId: vpc.VpcId,
				}

			}

			if isEligibleForNuke {
				// No specific filters to apply at this point, we can think about introducing
				// filtering with name tag in the future. In the initial version, we just getAll
				// without filtering.
				dhcpOptionIds = append(dhcpOptionIds, dhcpOption.DhcpOptionsId)
			}
		}
	}

	// checking the nukable permissions
	v.VerifyNukablePermissions(dhcpOptionIds, func(id *string) error {
		_, err := v.Client.DeleteDhcpOptions(v.Context, &ec2.DeleteDhcpOptionsInput{
			DhcpOptionsId: id,
			DryRun:        aws.Bool(true),
		})
		return err
	})

	return dhcpOptionIds, nil
}

func (v *EC2DhcpOption) disAssociatedAttachedVpcs(identifier *string) error {
	if option, ok := v.DhcpOptions[aws.ToString(identifier)]; ok {
		logging.Debugf("[disAssociatedAttachedVpcs] detaching the dhcp option %s from %v", aws.ToString(identifier), aws.ToString(option.VpcId))

		_, err := v.Client.AssociateDhcpOptions(v.Context, &ec2.AssociateDhcpOptionsInput{
			VpcId:         option.VpcId,
			DhcpOptionsId: aws.String("default"), // The ID of the DHCP options set, or default to associate no DHCP options with the VPC.
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("[disAssociatedAttachedVpcs] Success dhcp option %s detached from %v", aws.ToString(identifier), aws.ToString(option.VpcId))
	}

	return nil
}

func (v *EC2DhcpOption) nuke(identifier *string) error {

	err := v.disAssociatedAttachedVpcs(identifier)
	if err != nil {
		logging.Debugf("[disAssociatedAttachedVpcs] Failed %s", err)
		return errors.WithStackTrace(err)
	}

	err = nukeDhcpOption(v.Context, v.Client, identifier)
	if err != nil {
		logging.Debugf("[nukeDhcpOption] Failed %s", err)
		return errors.WithStackTrace(err)
	}
	return nil
}

func (v *EC2DhcpOption) nukeAll(identifiers []*string) error {
	for _, identifier := range identifiers {
		if nukable, reason := v.IsNukable(aws.ToString(identifier)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(identifier), reason)
			continue
		}

		err := v.nuke(identifier)
		if err != nil {
			logging.Debugf("Failed to delete DHCP option w/ err: %s.", err)
		} else {
			logging.Infof("Successfully deleted DHCP option %s.", pterm.Green(*identifier))
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: v.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}

func nukeDhcpOption(ctx context.Context, client EC2DhcpOptionAPI, option *string) error {
	logging.Debugf("Deleting DHCP Option %s", aws.ToString(option))

	_, err := client.DeleteDhcpOptions(ctx, &ec2.DeleteDhcpOptionsInput{
		DhcpOptionsId: option,
	})
	if err != nil {
		logging.Debugf("[Failed] Error deleting DHCP option %s: %s", aws.ToString(option), err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("[Ok] DHCP Option deleted successfully %s", aws.ToString(option))
	return nil
}
