package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/vpclattice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (network *VPCLatticeServiceNetwork) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	output, err := network.Client.ListServiceNetworksWithContext(network.Context, nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, item := range output.Items {

		if configObj.VPCLatticeServiceNetwork.ShouldInclude(config.ResourceValue{
			Name: item.Name,
			Time: item.CreatedAt,
		}) {
			ids = append(ids, item.Arn)
		}
	}

	return ids, nil
}

func (network *VPCLatticeServiceNetwork) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in region %s", network.ResourceServiceName(), network.Region)
		return nil

	}

	logging.Debugf("Deleting all %s in region %s", network.ResourceServiceName(), network.Region)

	deletedCount := 0
	for _, id := range identifiers {

		_, err := network.Client.DeleteServiceNetworkWithContext(network.Context, &vpclattice.DeleteServiceNetworkInput{
			ServiceNetworkIdentifier: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: network.ResourceServiceName(),
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedCount++
			logging.Debugf("Deleted %s: %s", network.ResourceServiceName(), aws.StringValue(id))
		}
	}

	logging.Debugf("[OK] %d %s(s) terminated in %s", deletedCount, network.ResourceServiceName(), network.Region)
	return nil
}
