package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (network *VPCLatticeService) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	output, err := network.Client.ListServices(network.Context, nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, item := range output.Items {

		if configObj.VPCLatticeService.ShouldInclude(config.ResourceValue{
			Name: item.Name,
			Time: item.CreatedAt,
		}) {
			ids = append(ids, item.Arn)
		}
	}

	return ids, nil
}

func (network *VPCLatticeService) nukeServiceAssociations(id *string) error {
	// list service associations
	associations, err := network.Client.ListServiceNetworkServiceAssociations(network.Context, &vpclattice.ListServiceNetworkServiceAssociationsInput{
		ServiceIdentifier: id,
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, item := range associations.Items {
		// list service associations
		_, err := network.Client.DeleteServiceNetworkServiceAssociation(network.Context, &vpclattice.DeleteServiceNetworkServiceAssociationInput{
			ServiceNetworkServiceAssociationIdentifier: item.Id,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}
	return nil
}

func (network *VPCLatticeService) nukeService(id *string) error {
	_, err := network.Client.DeleteService(network.Context, &vpclattice.DeleteServiceInput{
		ServiceIdentifier: id,
	})
	return err
}

func (network *VPCLatticeService) nuke(id *string) error {
	if err := network.nukeServiceAssociations(id); err != nil {
		return err
	}

	if err := network.waitUntilAllServiceAssociationDeleted(id); err != nil {
		return err
	}
	if err := network.nukeService(id); err != nil {
		return err
	}

	return nil
}
func (network *VPCLatticeService) waitUntilAllServiceAssociationDeleted(id *string) error {
	for i := 0; i < 10; i++ {
		output, err := network.Client.ListServiceNetworkServiceAssociations(network.Context, &vpclattice.ListServiceNetworkServiceAssociationsInput{
			ServiceIdentifier: id,
		})

		if err != nil {
			return errors.WithStackTrace(err)
		}
		if len(output.Items) == 0 {
			return nil
		}
		logging.Info("Waiting for service associations to be deleted...")
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timed out waiting for service associations to be successfully deleted")

}

func (network *VPCLatticeService) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in region %s", network.ResourceServiceName(), network.Region)
		return nil

	}

	logging.Debugf("Deleting all %s in region %s", network.ResourceServiceName(), network.Region)

	deletedCount := 0
	for _, id := range identifiers {

		err := network.nuke(id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: network.ResourceServiceName(),
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedCount++
			logging.Debugf("Deleted %s: %s", network.ResourceServiceName(), aws.ToString(id))
		}
	}

	logging.Debugf("[OK] %d %s(s) terminated in %s", deletedCount, network.ResourceServiceName(), network.Region)
	return nil
}
