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

func (dsg DBSubnetGroups) ResourceName() string {
	return "rds-subnet-group"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (dsg DBSubnetGroups) ResourceIdentifiers() []string {
	return dsg.InstanceNames
}

func (dsg DBSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (dsg DBSubnetGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := dsg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
