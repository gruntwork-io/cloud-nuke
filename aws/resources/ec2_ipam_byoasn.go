package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of IPAM Byoasns
func (byoasn *EC2IPAMByoasn) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}
	params := &ec2.DescribeIpamByoasnInput{
		MaxResults: &MaxResultCount,
	}

	output, err := byoasn.Client.DescribeIpamByoasn(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, out := range output.Byoasns {
		result = append(result, out.Asn)
	}

	// checking the nukable permissions
	byoasn.VerifyNukablePermissions(result, func(id *string) error {
		_, err := byoasn.Client.DisassociateIpamByoasn(&ec2.DisassociateIpamByoasnInput{
			Asn:    id,
			DryRun: awsgo.Bool(true),
		})
		return err
	})
	return result, nil
}

// Deletes all IPAMs Byoasns
func (byoasn *EC2IPAMByoasn) nukeAll(asns []*string) error {
	if len(asns) == 0 {
		logging.Debugf("No IPAM Byoasn to nuke in region %s", byoasn.Region)
		return nil
	}

	logging.Debugf("Deleting all IPAM Byoasn in region %s", byoasn.Region)
	var list []*string

	for _, id := range asns {
		if nukable, err := byoasn.IsNukable(awsgo.StringValue(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(id), err)
			continue
		}

		_, err := byoasn.Client.DisassociateIpamByoasn(&ec2.DisassociateIpamByoasnInput{
			Asn: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "IPAM Byoasn",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			list = append(list, id)
			logging.Debugf("Deleted IPAM Pool: %s", *id)
		}
	}

	logging.Debugf("[OK] %d IPAM Pool(s) deleted in %s", len(list), byoasn.Region)

	return nil
}
