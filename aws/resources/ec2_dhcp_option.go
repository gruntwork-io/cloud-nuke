package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
)

func (v *EC2DhcpOption) getAll(c context.Context, configObj config.Config) ([]*string, error) {
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
		_, err := v.Client.DeleteDhcpOptions(&ec2.DeleteDhcpOptionsInput{
			DhcpOptionsId: identifier,
		})
		if err != nil {
			pterm.Debug.Println(fmt.Sprintf("Failed to delete DHCP option w/ err: %s.", err))
			return errors.WithStackTrace(err)
		}
	}

	return nil
}
