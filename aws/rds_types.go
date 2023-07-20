package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBInstance struct {
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (instance DBInstance) ResourceName() string {
	return "rds-instance"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance DBInstance) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance DBInstance) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance DBInstance) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRdsInstances(session, awsgo.StringSlice(identifiers)); err != nil {
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
