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
	err := v.Client.DescribeDhcpOptionsPages(&ec2.DescribeDhcpOptionsInput{}, func(page *ec2.DescribeDhcpOptionsOutput, lastPage bool) bool {
		for _, dhcpOption := range page.DhcpOptions {
			// No specific filters to apply at this point, we can think about introducing
			// filtering with name tag in the future. In the initial version, we just getAll
			// without filtering.
			dhcpOptionIds = append(dhcpOptionIds, dhcpOption.DhcpOptionsId)
		}

		return !lastPage
	})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return dhcpOptionIds, nil
}

func (v *EC2DhcpOption) nukeAll(identifiers []*string) error {
	for _, identifier := range identifiers {

		err := nukeDhcpOption(v.Client, identifier)
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
