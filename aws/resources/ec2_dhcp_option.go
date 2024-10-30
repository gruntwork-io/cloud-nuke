package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
)

func (v *EC2DhcpOption) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var dhcpOptionIds []*string
	err := v.Client.DescribeDhcpOptionsPagesWithContext(v.Context, &ec2.DescribeDhcpOptionsInput{}, func(page *ec2.DescribeDhcpOptionsOutput, lastPage bool) bool {
		for _, dhcpOption := range page.DhcpOptions {
			var isEligibleForNuke = true

			// check the dhcp is attached with any vpc
			// if the vpc is default one, then omit the dhcp option id from result
			vpcs, err := v.Client.DescribeVpcsWithContext(v.Context, &ec2.DescribeVpcsInput{
				Filters: []*ec2.Filter{
					{
						Name:   awsgo.String("dhcp-options-id"),
						Values: []*string{dhcpOption.DhcpOptionsId},
					},
				},
			})
			if err != nil {
				logging.Debugf("[Failed] %s", err)
				continue
			}

			for _, vpc := range vpcs.Vpcs {
				// check the vpc is the default one then set isEligible false as we dont need to remove the dhcp option of default vpc
				if awsgo.BoolValue(vpc.IsDefault) {
					logging.Debugf("[Skipping] %s is attached with a default vpc %s", awsgo.StringValue(dhcpOption.DhcpOptionsId), awsgo.StringValue(vpc.VpcId))
					isEligibleForNuke = false
				}

				v.DhcpOptions[awsgo.StringValue(dhcpOption.DhcpOptionsId)] = DHCPOption{
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

		return !lastPage
	})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// checking the nukable permissions
	v.VerifyNukablePermissions(dhcpOptionIds, func(id *string) error {
		_, err := v.Client.DeleteDhcpOptionsWithContext(v.Context, &ec2.DeleteDhcpOptionsInput{
			DhcpOptionsId: id,
			DryRun:        awsgo.Bool(true),
		})
		return err
	})

	return dhcpOptionIds, nil
}

func (v *EC2DhcpOption) disAssociatedAttachedVpcs(identifier *string) error {
	if option, ok := v.DhcpOptions[awsgo.StringValue(identifier)]; ok {
		logging.Debugf("[disAssociatedAttachedVpcs] detaching the dhcp option %s from %v", awsgo.StringValue(identifier), awsgo.StringValue(option.VpcId))

		_, err := v.Client.AssociateDhcpOptionsWithContext(v.Context, &ec2.AssociateDhcpOptionsInput{
			VpcId:         option.VpcId,
			DhcpOptionsId: awsgo.String("default"), // The ID of the DHCP options set, or default to associate no DHCP options with the VPC.
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("[disAssociatedAttachedVpcs] Success dhcp option %s detached from %v", awsgo.StringValue(identifier), awsgo.StringValue(option.VpcId))
	}

	return nil
}

func (v *EC2DhcpOption) nuke(identifier *string) error {

	err := v.disAssociatedAttachedVpcs(identifier)
	if err != nil {
		logging.Debugf("[disAssociatedAttachedVpcs] Failed %s", err)
		return errors.WithStackTrace(err)
	}

	err = nukeDhcpOption(v.Client, identifier)
	if err != nil {
		logging.Debugf("[nukeDhcpOption] Failed %s", err)
		return errors.WithStackTrace(err)
	}
	return nil
}

func (v *EC2DhcpOption) nukeAll(identifiers []*string) error {
	for _, identifier := range identifiers {
		if nukable, reason := v.IsNukable(awsgo.StringValue(identifier)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(identifier), reason)
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
			Identifier:   awsgo.StringValue(identifier),
			ResourceType: v.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}

func nukeDhcpOption(client ec2iface.EC2API, option *string) error {
	logging.Debugf("Deleting DHCP Option %s", awsgo.StringValue(option))

	_, err := client.DeleteDhcpOptions(&ec2.DeleteDhcpOptionsInput{
		DhcpOptionsId: option,
	})
	if err != nil {
		logging.Debugf("[Failed] Error deleting DHCP option %s: %s", awsgo.StringValue(option), err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("[Ok] DHCP Option deleted successfully %s", awsgo.StringValue(option))
	return nil
}
