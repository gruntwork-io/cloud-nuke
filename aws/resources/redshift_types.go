package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/redshift/redshiftiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type RedshiftClusters struct {
	Client             redshiftiface.RedshiftAPI
	Region             string
	ClusterIdentifiers []string
}

func (rc *RedshiftClusters) Init(session *session.Session) {
	rc.Client = redshift.New(session)
}

func (rc *RedshiftClusters) ResourceName() string {
	return "redshift"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (rc *RedshiftClusters) ResourceIdentifiers() []string {
	return rc.ClusterIdentifiers
}

func (rc *RedshiftClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (rc *RedshiftClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := rc.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	rc.ClusterIdentifiers = awsgo.StringValueSlice(identifiers)
	return rc.ClusterIdentifiers, nil
}

// Nuke - nuke 'em all!!!
func (rc *RedshiftClusters) Nuke(identifiers []string) error {
	if err := rc.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
