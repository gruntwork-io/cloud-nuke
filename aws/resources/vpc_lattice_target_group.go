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

func (network *VPCLatticeTargetGroup) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	output, err := network.Client.ListTargetGroupsWithContext(network.Context, nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, item := range output.Items {

		if configObj.VPCLatticeTargetGroup.ShouldInclude(config.ResourceValue{
			Name: item.Name,
			Time: item.CreatedAt,
		}) {
			ids = append(ids, item.Arn)
			// also keep the complete info about the target groups as the target group assoiation needs to be nuked before removing it
			network.TargetGroups[aws.StringValue(item.Arn)] = item
		}
	}

	return ids, nil
}

func (network *VPCLatticeTargetGroup) nukeTargets(identifier *string) error {
	// list the targets associated on the target group
	output, err := network.Client.ListTargetsWithContext(network.Context, &vpclattice.ListTargetsInput{
		TargetGroupIdentifier: identifier,
	})
	if err != nil {
		logging.Debugf("[ListTargetsWithContext Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	var targets []*vpclattice.Target
	for _, target := range output.Items {
		targets = append(targets, &vpclattice.Target{
			Id: target.Id,
		})
	}

	if len(targets) > 0 {
		// before deleting the targets, we need to deregister the targets registered with it
		_, err = network.Client.DeregisterTargetsWithContext(network.Context, &vpclattice.DeregisterTargetsInput{
			TargetGroupIdentifier: identifier,
			Targets:               targets,
		})
		if err != nil {
			logging.Debugf("[DeregisterTargetsWithContext Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

func (network *VPCLatticeTargetGroup) nuke(identifier *string) error {

	var err error
	err = network.nukeTargets(identifier)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// delete the target group
	_, err = network.Client.DeleteTargetGroupWithContext(network.Context, &vpclattice.DeleteTargetGroupInput{
		TargetGroupIdentifier: identifier,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
func (network *VPCLatticeTargetGroup) nukeAll(identifiers []*string) error {
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
