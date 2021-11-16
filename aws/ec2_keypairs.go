package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllEc2KeyPairs(session *session.Session, region string) ([]*string, error) {
	svc := ec2.New(session)

	result, err := svc.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, keypair := range result.KeyPairs {
		ids = append(ids, keypair.KeyPairId)
	}

	return ids, nil
}

func deleteKeyPair(svc *ec2.EC2, keyPairId *string) error {
	_, err := svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyPairId: keyPairId,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func nukeAllEc2KeyPairs(session *session.Session, keypairIds []*string) error {
	if len(keypairIds) == 0 {
		logging.Logger.Info("No EC2 Key Pairs to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all EC2 Key Pairs")

	deletedKeyPairs := 0
	svc := ec2.New(session)
	multiErr := new(multierror.Error)

	for _, keypair := range keypairIds {
		if err := deleteKeyPair(svc, keypair); err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedKeyPairs++
			logging.Logger.Infof("Deleted EC2 KeyPair: %s", keypair)
		}
	}

	logging.Logger.Infof("[OK] %d EC2 KeyPair(s) terminated", deletedKeyPairs)

	return nil
}
