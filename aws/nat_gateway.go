package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func waitUntilNatGatewayDeleted(svc *ec2.EC2, input *ec2.DescribeNatGatewaysInput) error {
	for i := 0; i < 30; i++ {
		result, err := svc.DescribeNatGateways(input)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "NatGatewayNotFound" {
				return nil
			}

			return err
		}

		if result.NatGateways != nil && len(result.NatGateways) > 0 {
			if *result.NatGateways[0].State == "deleted" {
				return nil
			}
		}

		logging.Logger.Debug("NAT Gateway still not deleted. Will sleep for 5 seconds and check again")
		time.Sleep(5 * time.Second)
	}

	return NatGatewayDeleteError{}
}

// Returns a formatted string of NAT Gateways Ids
func getAllNatGateways(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{
				Name: awsgo.String("state"),
				Values: []*string{
					awsgo.String("available"),
				},
			},
		},
	}

	result, err := svc.DescribeNatGateways(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var natGatewaysIds []*string
	for _, natGateway := range result.NatGateways {
		if excludeAfter.After(*natGateway.CreateTime) {
			natGatewaysIds = append(natGatewaysIds, natGateway.NatGatewayId)
		}
	}

	return natGatewaysIds, nil
}

// Deletes all NAT Gateways
func nukeAllNatGateways(session *session.Session, natGatewayIds []*string) error {
	svc := ec2.New(session)

	if len(natGatewayIds) == 0 {
		logging.Logger.Infof("No NAT Gateways to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all NAT Gateways in region %s", *session.Config.Region)
	var deletedNatGatewayIDs []*string

	for _, natGatewayID := range natGatewayIds {
		params := &ec2.DeleteNatGatewayInput{
			NatGatewayId: natGatewayID,
		}

		_, err := svc.DeleteNatGateway(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedNatGatewayIDs = append(deletedNatGatewayIDs, natGatewayID)
			logging.Logger.Infof("Deleted NAT Gateway: %s", *natGatewayID)
		}
	}

	if len(deletedNatGatewayIDs) > 0 {
		for _, natGatewayID := range deletedNatGatewayIDs {
			err := waitUntilNatGatewayDeleted(svc, &ec2.DescribeNatGatewaysInput{
				NatGatewayIds: []*string{natGatewayID},
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Infof("[OK] %d NAT Gateway(s) deleted in %s", len(deletedNatGatewayIDs), *session.Config.Region)
	return nil
}
