package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/redshift/redshiftiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type RedshiftClusters struct {
	Client             redshiftiface.RedshiftAPI
	Region             string
	ClusterIdentifiers []string
}

func (cluster RedshiftClusters) ResourceName() string {
	return "redshift"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (cluster RedshiftClusters) ResourceIdentifiers() []string {
	return cluster.ClusterIdentifiers
}

func (cluster RedshiftClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (cluster RedshiftClusters) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRedshiftClusters(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
