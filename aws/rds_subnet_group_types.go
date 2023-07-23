package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBSubnetGroups struct {
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (instance DBSubnetGroups) ResourceName() string {
	return "rds-subnet-group"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance DBSubnetGroups) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance DBSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance DBSubnetGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRdsDbSubnetGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
