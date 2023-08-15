package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// Snapshots - represents all user owned Snapshots
type Snapshots struct {
	Client      ec2iface.EC2API
	Region      string
	SnapshotIds []string
}

func (s *Snapshots) Init(session *session.Session) {
	s.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (s *Snapshots) ResourceName() string {
	return "snap"
}

// ResourceIdentifiers - The Snapshot snapshot ids
func (s *Snapshots) ResourceIdentifiers() []string {
	return s.SnapshotIds
}

func (s *Snapshots) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (s *Snapshots) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := s.getAll(configObj)
	if err != nil {
		return nil, err
	}

	s.SnapshotIds = awsgo.StringValueSlice(identifiers)
	return s.SnapshotIds, nil
}

// Nuke - nuke 'em all!!!
func (s *Snapshots) Nuke(identifiers []string) error {
	if err := s.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
