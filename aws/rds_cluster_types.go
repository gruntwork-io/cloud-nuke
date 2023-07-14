package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBClusters struct {
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (instance DBClusters) ResourceName() string {
	return "rds"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance DBClusters) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance DBClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance DBClusters) Nuke(session *session.Session, identifiers []string) error {
	if err := instance.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
