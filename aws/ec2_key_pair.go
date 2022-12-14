package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

// getAllEc2KeyPairs extracts the list of existing ec2 key pairs.
func getAllEc2KeyPairs(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)

	result, err := svc.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, keyPair := range result.KeyPairs {
		if shouldIncludeEc2KeyPair(keyPair, excludeAfter, configObj) {
			ids = append(ids, keyPair.KeyPairId)
		}
	}

	return ids, nil
}

func shouldIncludeEc2KeyPair(keyPairInfo *ec2.KeyPairInfo, excludeAfter time.Time, configObj config.Config) bool {
	if keyPairInfo == nil || keyPairInfo.KeyName == nil {
		return false
	}

	if keyPairInfo.CreateTime != nil && excludeAfter.Before(*keyPairInfo.CreateTime) {
		return false
	}

	return config.ShouldInclude(
		*keyPairInfo.KeyName,
		configObj.EC2KeyPairs.IncludeRule.NamesRegExp,
		configObj.EC2KeyPairs.ExcludeRule.NamesRegExp,
	)
}

// deleteKeyPair is a helper method that deletes the given ec2 key pair.
func deleteKeyPair(svc *ec2.EC2, keyPairId *string) error {
	params := &ec2.DeleteKeyPairInput{
		KeyPairId: keyPairId,
	}

	_, err := svc.DeleteKeyPair(params)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// nukeAllEc2KeyPairs attempts to delete given ec2 key pair IDs.
func nukeAllEc2KeyPairs(session *session.Session, keypairIds []*string) error {
	svc := ec2.New(session)

	if len(keypairIds) == 0 {
		logging.Logger.Infof("No EC2 key pairs to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Terminating all EC2 key pairs in region %s", *session.Config.Region)

	deletedKeyPairs := 0
	var multiErr *multierror.Error
	for _, keypair := range keypairIds {
		if err := deleteKeyPair(svc, keypair); err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multiErr = multierror.Append(multiErr, err)
		} else {
			deletedKeyPairs++
			logging.Logger.Infof("Deleted EC2 KeyPair: %s", *keypair)
		}
	}

	logging.Logger.Infof("[OK] %d EC2 KeyPair(s) terminated", deletedKeyPairs)
	return multiErr.ErrorOrNil()
}
