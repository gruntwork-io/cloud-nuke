package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBCluster struct {
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (instance DBCluster) ResourceName() string {
	return "rds-cluster"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance DBCluster) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance DBCluster) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance DBCluster) Nuke(session *session.Session, identifiers []string) error {
	if err := instance.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
