package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBInstances struct {
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (di DBInstances) ResourceName() string {
	return "rds"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (di DBInstances) ResourceIdentifiers() []string {
	return di.InstanceNames
}

func (di DBInstances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (di DBInstances) Nuke(session *session.Session, identifiers []string) error {
	if err := di.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type RdsDeleteError struct {
	name string
}

func (e RdsDeleteError) Error() string {
	return "RDS DB Instance:" + e.name + "was not deleted"
}
